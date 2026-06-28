package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"net"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
)

func HermesHandler(msg []byte) error {
	var hermesPayload structs.WsHermesPayload
	if err := json.Unmarshal(msg, &hermesPayload); err != nil {
		return fmt.Errorf("couldn't unmarshal Hermes payload: %v", err)
	}
	switch hermesPayload.Payload.Action {
	case "install":
		go handleHermesInstall(hermesPayload)
	case "update":
		go handleHermesUpdate(hermesPayload)
	case "toggle":
		go handleHermesToggle(hermesPayload)
	case "save":
		go handleHermesSave(hermesPayload)
	case "restart":
		go handleHermesRestart()
	default:
		return fmt.Errorf("unrecognized Hermes action: %v", hermesPayload.Payload.Action)
	}
	return nil
}

func handleHermesInstall(hermesPayload structs.WsHermesPayload) {
	clearHermesError()
	docker.HermesTransBus <- structs.Event{Type: "install", Data: "preparing"}
	defer clearHermesTransition("install")
	if err := config.LoadHermesConfig(); err != nil {
		failHermesTransition("install", err)
		return
	}
	hermesConf := config.HermesConf()
	if err := applyHermesPayload(hermesPayload.Payload, &hermesConf); err != nil {
		failHermesTransition("install", err)
		return
	}
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		failHermesTransition("install", err)
		return
	}
	image := docker.HermesImageOrDefault(hermesConf.Image)
	zap.L().Info(fmt.Sprintf("Installing Hermes image %s", image))
	if err := docker.PullImageByRefWithProgress(image, func(status string) {
		docker.HermesTransBus <- structs.Event{Type: "install", Data: status}
	}); err != nil {
		failHermesTransition("install", err)
		return
	}
	zap.L().Info(fmt.Sprintf("Hermes image %s installed", image))
	docker.HermesTransBus <- structs.Event{Type: "install", Data: "success"}
}

func handleHermesUpdate(hermesPayload structs.WsHermesPayload) {
	clearHermesError()
	docker.HermesTransBus <- structs.Event{Type: "install", Data: "preparing"}
	defer clearHermesTransition("install")
	if err := config.LoadHermesConfig(); err != nil {
		failHermesTransition("install", err)
		return
	}
	hermesConf := config.HermesConf()
	wasEnabled := hermesConf.Enabled
	if err := applyHermesPayload(hermesPayload.Payload, &hermesConf); err != nil {
		failHermesTransition("install", err)
		return
	}
	versionServerImage, err := docker.HermesVersionServerImage()
	if err != nil {
		failHermesTransition("install", err)
		return
	}
	hermesConf.Image = versionServerImage
	docker.HermesTransBus <- structs.Event{Type: "install", Data: "removing-container"}
	stopAndDeleteHermes(false)
	zap.L().Info(fmt.Sprintf("Updating Hermes image to %s", versionServerImage))
	if err := docker.PullImageByRefWithProgress(versionServerImage, func(status string) {
		docker.HermesTransBus <- structs.Event{Type: "install", Data: status}
	}); err != nil {
		failHermesTransition("install", err)
		return
	}
	if wasEnabled {
		docker.HermesTransBus <- structs.Event{Type: "install", Data: "validating"}
		if err := validateRunnableHermes(hermesConf); err != nil {
			failHermesTransition("install", err)
			return
		}
		docker.HermesTransBus <- structs.Event{Type: "install", Data: "fetching-code"}
		if err := refreshHermesAccessCode(&hermesConf); err != nil {
			failHermesTransition("install", err)
			return
		}
	}
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		failHermesTransition("install", err)
		return
	}
	if wasEnabled {
		docker.HermesTransBus <- structs.Event{Type: "install", Data: "starting"}
		if err := recreateHermesContainer(); err != nil {
			failHermesTransition("install", err)
			return
		}
	}
	zap.L().Info(fmt.Sprintf("Hermes image %s updated", versionServerImage))
	docker.HermesTransBus <- structs.Event{Type: "install", Data: "success"}
}

func handleHermesToggle(hermesPayload structs.WsHermesPayload) {
	clearHermesError()
	docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "loading"}
	defer clearHermesTransition("toggle")
	if err := config.LoadHermesConfig(); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	hermesConf := config.HermesConf()
	if hermesConf.Enabled {
		docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "stopping"}
		hermesConf.Enabled = false
		hermesConf.AccessCode = ""
		if err := config.UpdateHermesConfig(hermesConf); err != nil {
			failHermesTransition("toggle", err)
			return
		}
		stopAndDeleteHermes(false)
		docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "success"}
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "validating"}
	if err := applyHermesPayload(hermesPayload.Payload, &hermesConf); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	if err := validateRunnableHermes(hermesConf); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "fetching-code"}
	if err := refreshHermesAccessCode(&hermesConf); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	hermesConf.Enabled = true
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "starting"}
	if err := recreateHermesContainer(); err != nil {
		failHermesTransition("toggle", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "toggle", Data: "success"}
}

func handleHermesSave(hermesPayload structs.WsHermesPayload) {
	clearHermesError()
	docker.HermesTransBus <- structs.Event{Type: "save", Data: "saving"}
	defer clearHermesTransition("save")
	if err := config.LoadHermesConfig(); err != nil {
		failHermesTransition("save", err)
		return
	}
	hermesConf := config.HermesConf()
	if err := applyHermesPayload(hermesPayload.Payload, &hermesConf); err != nil {
		failHermesTransition("save", err)
		return
	}
	if hermesConf.Enabled {
		docker.HermesTransBus <- structs.Event{Type: "save", Data: "validating"}
		if err := validateRunnableHermes(hermesConf); err != nil {
			failHermesTransition("save", err)
			return
		}
		docker.HermesTransBus <- structs.Event{Type: "save", Data: "fetching-code"}
		if err := refreshHermesAccessCode(&hermesConf); err != nil {
			failHermesTransition("save", err)
			return
		}
	} else {
		hermesConf.AccessCode = ""
	}
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		failHermesTransition("save", err)
		return
	}
	if hermesConf.Enabled {
		docker.HermesTransBus <- structs.Event{Type: "save", Data: "restarting"}
		if err := recreateHermesContainer(); err != nil {
			failHermesTransition("save", err)
			return
		}
	}
	docker.HermesTransBus <- structs.Event{Type: "save", Data: "success"}
}

func handleHermesRestart() {
	clearHermesError()
	docker.HermesTransBus <- structs.Event{Type: "restart", Data: "validating"}
	defer clearHermesTransition("restart")
	if err := config.LoadHermesConfig(); err != nil {
		failHermesTransition("restart", err)
		return
	}
	hermesConf := config.HermesConf()
	if !hermesConf.Enabled {
		failHermesTransition("restart", fmt.Errorf("Hermes is not enabled"))
		return
	}
	if err := validateRunnableHermes(hermesConf); err != nil {
		failHermesTransition("restart", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "restart", Data: "fetching-code"}
	if err := refreshHermesAccessCode(&hermesConf); err != nil {
		failHermesTransition("restart", err)
		return
	}
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		failHermesTransition("restart", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "restart", Data: "recreating"}
	if err := recreateHermesContainer(); err != nil {
		failHermesTransition("restart", err)
		return
	}
	docker.HermesTransBus <- structs.Event{Type: "restart", Data: "success"}
}

func applyHermesPayload(payload structs.WsHermesAction, hermesConf *structs.HermesConfig) error {
	if ship := docker.NormalizeHermesShip(payload.Ship); ship != "" {
		hermesConf.Ship = ship
	}
	if owner := docker.NormalizeHermesShip(payload.Owner); owner != "" {
		hermesConf.Owner = owner
	}
	if payload.Port > 0 {
		if payload.Port > 65535 {
			return fmt.Errorf("invalid Hermes port %d", payload.Port)
		}
		hermesConf.Port = payload.Port
	}
	if hermesConf.Port <= 0 {
		port, err := nextHermesPort()
		if err != nil {
			return err
		}
		hermesConf.Port = port
	}
	if image := strings.TrimSpace(payload.Image); image != "" {
		if !isPinnedImageRef(image) {
			return fmt.Errorf("Hermes image must be pinned by non-latest tag or sha256 digest")
		}
		hermesConf.Image = image
	}
	if strings.TrimSpace(hermesConf.Image) == "" {
		hermesConf.Image = docker.HermesImageOrDefault("")
	}
	if !isPinnedImageRef(hermesConf.Image) {
		return fmt.Errorf("Hermes image must be pinned by non-latest tag or sha256 digest")
	}
	if provider := strings.TrimSpace(payload.ModelProvider); provider != "" {
		normalizedProvider := docker.NormalizeHermesModelProvider(provider)
		if normalizedProvider == "" {
			return fmt.Errorf("unsupported Hermes provider %q", provider)
		}
		if normalizedProvider != hermesConf.ModelProvider && strings.TrimSpace(payload.ProviderAPIKey) == "" {
			hermesConf.ProviderAPIKey = ""
		}
		hermesConf.ModelProvider = normalizedProvider
	}
	if model := strings.TrimSpace(payload.Model); model != "" {
		hermesConf.Model = model
	}
	if providerAPIKey := strings.TrimSpace(payload.ProviderAPIKey); providerAPIKey != "" {
		hermesConf.ProviderAPIKey = providerAPIKey
	}
	if webProvider := strings.TrimSpace(payload.WebProvider); webProvider != "" {
		normalizedWebProvider := docker.NormalizeHermesWebProvider(webProvider)
		if normalizedWebProvider == "" {
			return fmt.Errorf("unsupported Hermes web provider %q", webProvider)
		}
		if normalizedWebProvider != hermesConf.WebProvider && strings.TrimSpace(payload.WebAPIKey) == "" {
			hermesConf.WebAPIKey = ""
		}
		if normalizedWebProvider != hermesConf.WebProvider && strings.TrimSpace(payload.WebURL) == "" {
			hermesConf.WebURL = ""
		}
		hermesConf.WebProvider = normalizedWebProvider
	} else if payload.Action == "save" || payload.Action == "toggle" || payload.Action == "install" || payload.Action == "update" {
		hermesConf.WebProvider = ""
		hermesConf.WebAPIKey = ""
		hermesConf.WebURL = ""
	}
	if webAPIKey := strings.TrimSpace(payload.WebAPIKey); webAPIKey != "" {
		hermesConf.WebAPIKey = webAPIKey
	}
	if webURL := strings.TrimSpace(payload.WebURL); webURL != "" {
		hermesConf.WebURL = webURL
	}
	if webProvider, ok := docker.HermesWebProviderConfig(hermesConf.WebProvider); ok {
		if webProvider.APIKeyEnv == "" {
			hermesConf.WebAPIKey = ""
		}
		if webProvider.URLEnv == "" {
			hermesConf.WebURL = ""
		}
	}
	if payload.Action == "save" || payload.Action == "toggle" || payload.Action == "install" {
		hermesConf.APIEnabled = payload.APIEnabled
	}
	if apiKey := strings.TrimSpace(payload.APIKey); apiKey != "" {
		hermesConf.APIKey = apiKey
	}
	if strings.TrimSpace(hermesConf.ModelProvider) == "" {
		hermesConf.ModelProvider = docker.DefaultHermesModelProvider
	}
	if strings.TrimSpace(hermesConf.Model) == "" {
		hermesConf.Model = docker.DefaultHermesModel
	}
	if strings.TrimSpace(hermesConf.HermesVersion) == "" {
		hermesConf.HermesVersion = docker.DefaultHermesVersion
	}
	if strings.TrimSpace(hermesConf.HermesAgentRef) == "" {
		hermesConf.HermesAgentRef = docker.DefaultHermesAgentRef
	}
	if strings.TrimSpace(hermesConf.TlonAdapterVersion) == "" {
		hermesConf.TlonAdapterVersion = docker.DefaultHermesTlonAdapterVersion
	}
	if strings.TrimSpace(hermesConf.TlonAdapterRef) == "" {
		hermesConf.TlonAdapterRef = docker.DefaultHermesTlonAdapterRef
	}
	if strings.TrimSpace(hermesConf.WebProvider) != "" && docker.NormalizeHermesWebProvider(hermesConf.WebProvider) == "" {
		return fmt.Errorf("unsupported Hermes web provider %q", hermesConf.WebProvider)
	}
	return nil
}

func validateRunnableHermes(hermesConf structs.HermesConfig) error {
	ship := strings.TrimPrefix(docker.NormalizeHermesShip(hermesConf.Ship), "~")
	if ship == "" {
		return fmt.Errorf("Hermes ship is required")
	}
	if docker.NormalizeHermesShip(hermesConf.Owner) == "" {
		return fmt.Errorf("Hermes owner is required")
	}
	if !pierExists(ship) {
		return fmt.Errorf("Hermes ship %s is not managed by GroundSeg", docker.NormalizeHermesShip(ship))
	}
	apiKeyEnv := docker.HermesProviderAPIKeyEnv(hermesConf.ModelProvider)
	if apiKeyEnv == "" {
		return fmt.Errorf("unsupported Hermes provider %q", hermesConf.ModelProvider)
	}
	if strings.TrimSpace(hermesConf.ProviderAPIKey) == "" {
		return fmt.Errorf("Hermes provider API key is required for %s", docker.HermesModelProviderOrDefault(hermesConf.ModelProvider))
	}
	if webProvider := docker.NormalizeHermesWebProvider(hermesConf.WebProvider); webProvider != "" {
		webProviderConfig, ok := docker.HermesWebProviderConfig(webProvider)
		if !ok {
			return fmt.Errorf("unsupported Hermes web provider %q", hermesConf.WebProvider)
		}
		if webProviderConfig.APIKeyEnv != "" && strings.TrimSpace(hermesConf.WebAPIKey) == "" {
			return fmt.Errorf("Hermes web API key is required for %s", webProvider)
		}
		if webProviderConfig.URLEnv != "" && strings.TrimSpace(hermesConf.WebURL) == "" {
			return fmt.Errorf("Hermes web URL is required for %s", webProvider)
		}
	}
	if hermesConf.APIEnabled && strings.TrimSpace(hermesConf.APIKey) == "" {
		return fmt.Errorf("Hermes API key is required when the API server is enabled")
	}
	installed, err := docker.ImageRefExists(docker.HermesImageOrDefault(hermesConf.Image))
	if err != nil {
		return fmt.Errorf("failed to inspect Hermes image: %v", err)
	}
	if !installed {
		return fmt.Errorf("install the Hermes image before enabling Hermes")
	}
	return nil
}

func pierExists(patp string) bool {
	return slices.Contains(config.Conf().Piers, patp)
}

func refreshHermesAccessCode(hermesConf *structs.HermesConfig) error {
	patp := strings.TrimPrefix(docker.NormalizeHermesShip(hermesConf.Ship), "~")
	statuses, err := docker.GetShipStatus([]string{patp})
	if err != nil {
		return fmt.Errorf("failed to get ship status for Hermes %s: %v", patp, err)
	}
	status, exists := statuses[patp]
	if !exists || !strings.Contains(status, "Up") {
		return fmt.Errorf("ship %s must be running before Hermes can start", patp)
	}
	click.ClearLusCode(patp)
	code, err := click.GetLusCode(patp)
	if err != nil {
		return fmt.Errorf("failed to fetch +code for Hermes %s: %v", patp, err)
	}
	zap.L().Info(fmt.Sprintf("Fetched fresh +code for Hermes %s", patp))
	hermesConf.AccessCode = code
	return nil
}

func recreateHermesContainer() error {
	zap.L().Info("Recreating Hermes container")
	stopAndDeleteHermes(false)
	zap.L().Info("Starting Hermes container")
	info, err := docker.StartContainer(docker.HermesContainerName, "hermes")
	if err != nil {
		return fmt.Errorf("couldn't start Hermes: %v", err)
	}
	config.UpdateContainerState(docker.HermesContainerName, info)
	zap.L().Info("Hermes container started")
	return nil
}

func restartHermesForShipIfEnabled(patp string) {
	if err := config.LoadHermesConfig(); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to load Hermes config for ship restart check: %v", err))
		return
	}
	hermesConf := config.HermesConf()
	if !hermesConf.Enabled || strings.TrimPrefix(docker.NormalizeHermesShip(hermesConf.Ship), "~") != patp {
		return
	}
	go handleHermesRestart()
}

func disableHermesIfAssignedTo(patp string) {
	if err := config.LoadHermesConfig(); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to load Hermes config for ship delete check: %v", err))
		return
	}
	hermesConf := config.HermesConf()
	if strings.TrimPrefix(docker.NormalizeHermesShip(hermesConf.Ship), "~") != patp {
		return
	}
	hermesConf.Enabled = false
	hermesConf.AccessCode = ""
	if err := config.UpdateHermesConfig(hermesConf); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to disable Hermes for deleted ship %s: %v", patp, err))
	}
	stopAndDeleteHermes(false)
}

func stopAndDeleteHermes(deleteVolume bool) {
	if existing, err := docker.FindContainer(docker.HermesContainerName); err == nil && existing != nil {
		zap.L().Info("Stopping existing Hermes container")
		if existing.State == "running" {
			if err := docker.StopContainerByName(docker.HermesContainerName); err != nil {
				zap.L().Warn(fmt.Sprintf("Couldn't stop Hermes container: %v", err))
			}
		}
		if err := docker.DeleteContainer(docker.HermesContainerName); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't delete Hermes container: %v", err))
		}
	}
	if deleteVolume {
		if err := docker.DeleteVolume(docker.HermesDataVolumeName); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't delete Hermes volume: %v", err))
		}
		if err := docker.DeleteVolume(docker.HermesWorkspaceVolumeName); err != nil {
			zap.L().Warn(fmt.Sprintf("Couldn't delete Hermes workspace volume: %v", err))
		}
	}
	config.DeleteContainerState(docker.HermesContainerName)
}

func nextHermesPort() (int, error) {
	for port := docker.DefaultHermesDashboardHostPort; port <= 19999; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no open Hermes dashboard port found")
}

func isPinnedImageRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if strings.Contains(ref, "@sha256:") {
		return true
	}
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon <= lastSlash {
		return false
	}
	tag := strings.TrimSpace(ref[lastColon+1:])
	return tag != "" && tag != "latest"
}

func failHermesTransition(kind string, err error) {
	zap.L().Error(fmt.Sprintf("Hermes %s failed: %v", kind, err))
	docker.HermesTransBus <- structs.Event{Type: "error", Data: err.Error()}
	docker.HermesTransBus <- structs.Event{Type: kind, Data: "error"}
}

func clearHermesError() {
	docker.HermesTransBus <- structs.Event{Type: "error", Data: nil}
}

func clearHermesTransition(kind string) {
	time.Sleep(2 * time.Second)
	docker.HermesTransBus <- structs.Event{Type: kind, Data: nil}
}
