package rectify

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

type UrbitTransitionApplier struct{}

func (applier UrbitTransitionApplier) Apply(current *structs.AuthBroadcast, event structs.UrbitTransition) error {
	urbitStruct, exists := current.Urbits[event.Patp]
	if !exists {
		return nil
	}
	if !setUrbitTransition(&urbitStruct.Transition, event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
		return nil
	}
	current.Urbits[event.Patp] = urbitStruct
	return nil
}

type StartramTransitionService struct {
	runtime RectifyRuntime
}

func NewStartramTransitionService(runtime RectifyRuntime) StartramTransitionService {
	return StartramTransitionService{
		runtime: runtime,
	}
}

func (service StartramTransitionService) Apply(current *structs.AuthBroadcast, event structs.Event) error {
	if !setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data) {
		return nil
	}
	return service.applyTransitionCompletion(current, transition.EventType(event.Type), event.Data)
}

func (service StartramTransitionService) applyTransitionCompletion(current *structs.AuthBroadcast, eventType transition.EventType, eventData any) error {
	switch eventType {
	case transition.StartramTransitionEndpoint:
		if eventData != transition.StartramTransitionComplete {
			return nil
		}
		settings, err := service.runtime.StartramSettingsSnapshot()
		if err != nil {
			return fmt.Errorf("loading startram settings: %w", err)
		}
		current.Profile.Startram.Info.Endpoint = settings.EndpointURL
	case transition.StartramTransitionRegister:
		if eventData != transition.StartramTransitionComplete {
			return nil
		}
		settings, err := service.runtime.StartramSettingsSnapshot()
		if err != nil {
			return fmt.Errorf("loading startram settings: %w", err)
		}
		current.Profile.Startram.Info.Running = settings.WgOn
		containerState, exists := service.runtime.GetContainerStateFn()[string(transition.ContainerTypeWireguard)]
		if exists {
			running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
			current.Profile.Startram.Info.Running = running
			if err := service.runtime.UpdateConfig(config.WithWgOn(running)); err != nil {
				return err
			}
		}
		current.Profile.Startram.Info.Registered = settings.WgRegistered
	}
	return nil
}

type StartramRetrieveReconciler struct {
	runtime RectifyRuntime
	syncer  *urbitConfigSyncService
}

func NewStartramRetrieveReconciler(runtime RectifyRuntime) *StartramRetrieveReconciler {
	return &StartramRetrieveReconciler{
		runtime: runtime,
		syncer:  &urbitConfigSyncService{runtime: runtime},
	}
}

func (reconciler *StartramRetrieveReconciler) Reconcile(current *structs.AuthBroadcast) error {
	runtime := reconciler.runtime
	startramSettings, err := runtime.StartramSettingsSnapshot()
	if err != nil {
		return err
	}
	startramConfig, err := runtime.StartramConfig()
	if err != nil {
		return err
	}
	var reconcileErr error
	for patp := range runtime.UrbitConfAllFn() {
		modified := false
		serviceCreated := true
		local, err := reconciler.syncer.loadAndRefresh(patp)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Retrieve: unable to refresh urbit config for %s: %v", patp, err))
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("unable to refresh urbit config for %s: %w", patp, err))
			publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
			continue
		}
		persistNetwork := false
		persistWeb := false

		found := false
		for _, remote := range startramConfig.Subdomains {
			endpointUrl := strings.Split(startramSettings.EndpointURL, ".")
			if len(endpointUrl) < 2 {
				continue
			}
			rootUrl := strings.Join(endpointUrl[1:len(endpointUrl)], ".")
			if patp+"."+rootUrl == remote.URL {
				found = true
				break
			}
		}
		if !found {
			zap.L().Info(fmt.Sprintf("Registering missing StarTram service for %v", patp))
			startram.SvcCreate(patp, "urbit")
			startram.SvcCreate("s3."+patp, "minio")
		}

		for _, remote := range startramConfig.Subdomains {
			if remote.Status == string(transition.StartramServiceStatusCreating) {
				serviceCreated = false
			}
			parts := strings.Split(remote.URL, ".")
			if len(parts) < 2 {
				continue
			}
			if reconciler.reconcileUrbitWebService(patp, remote, &local, &modified, &persistWeb, &persistNetwork) {
				continue
			}
			if reconciler.reconcileUrbitNetworkServices(patp, remote, parts, &local, &modified, &persistNetwork) {
				continue
			}
		}

		if modified {
			if persistWeb {
				if err := reconciler.syncer.updateWebConfig(patp, &local); err != nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s web config updates: %v", patp, err))
					reconcileErr = errors.Join(reconcileErr, err)
				}
			}

			if persistNetwork {
				if err := reconciler.syncer.updateNetworkConfig(patp, &local); err != nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s network config updates: %v", patp, err))
					reconcileErr = errors.Join(reconcileErr, err)
				}
			}
		}
		publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
		publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
	}
	return reconcileErr
}

type urbitConfigSyncService struct {
	runtime RectifyRuntime
}

func (service *urbitConfigSyncService) loadAndRefresh(patp string) (structs.UrbitDocker, error) {
	if err := service.runtime.LoadUrbitConfigFn(patp); err != nil {
		return structs.UrbitDocker{}, err
	}
	local := service.runtime.UrbitConfFn(patp)
	return local, nil
}

func (service *urbitConfigSyncService) updateWebConfig(patp string, local *structs.UrbitDocker) error {
	return service.runtime.UpdateUrbitSectionFn(patp, config.UrbitConfigSectionWeb, func(webConfig *structs.UrbitWebConfig) error {
		webConfig.CustomUrbitWeb = local.CustomUrbitWeb
		return nil
	})
}

func (service *urbitConfigSyncService) updateNetworkConfig(patp string, local *structs.UrbitDocker) error {
	return service.runtime.UpdateUrbitSectionFn(patp, config.UrbitConfigSectionNetwork, func(networkConfig *structs.UrbitNetworkConfig) error {
		networkConfig.WgHTTPPort = local.WgHTTPPort
		networkConfig.WgAmesPort = local.WgAmesPort
		networkConfig.WgS3Port = local.WgS3Port
		networkConfig.WgConsolePort = local.WgConsolePort
		networkConfig.WgURL = local.WgURL
		networkConfig.Network = local.Network
		return nil
	})
}

func (reconciler *StartramRetrieveReconciler) reconcileUrbitWebService(patp string, remote structs.Subdomain, local *structs.UrbitDocker, modified *bool, persistWeb *bool, persistNetwork *bool) bool {
	parts := strings.Split(remote.URL, ".")
	if len(parts) < 2 {
		return false
	}
	subd := parts[0]
	if subd != patp || remote.SvcType != string(transition.StartramServiceTypeUrbitWeb) || remote.Status != string(transition.StartramServiceStatusOk) {
		return false
	}

	if remote.Alias == "null" && local.CustomUrbitWeb != "" {
		zap.L().Debug(fmt.Sprintf("Retrieve: Resetting %v alias", patp))
		local.CustomUrbitWeb = ""
		*persistWeb = true
		*modified = true
	} else if remote.Alias != local.CustomUrbitWeb {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v alias to %v", patp, remote.Alias))
		local.CustomUrbitWeb = remote.Alias
		*persistWeb = true
		*modified = true
	}
	if remote.Port != local.WgHTTPPort {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v WG port to %v", patp, remote.Port))
		local.WgHTTPPort = remote.Port
		*persistNetwork = true
		*modified = true
	}
	if remote.URL != local.WgURL {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v URL to %v", patp, remote.URL))
		local.WgURL = remote.URL
		*persistNetwork = true
		*modified = true
	}
	return true
}

func (reconciler *StartramRetrieveReconciler) reconcileUrbitNetworkServices(patp string, remote structs.Subdomain, urlParts []string, local *structs.UrbitDocker, modified *bool, persistNetwork *bool) bool {
	nestd := ""
	if len(urlParts) >= 2 {
		nestd = strings.Join(urlParts[:2], ".")
	}
	switch {
	case nestd == "ames."+patp && remote.SvcType == string(transition.StartramServiceTypeUrbitAmes) && remote.Status == string(transition.StartramServiceStatusOk):
		if remote.Port != local.WgAmesPort {
			zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v ames port to %v", patp, remote.Port))
			local.WgAmesPort = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	case nestd == "s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinio) && remote.Status == string(transition.StartramServiceStatusOk):
		if remote.Port != local.WgS3Port {
			zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v minio port to %v", patp, remote.Port))
			local.WgS3Port = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	}

	consd := ""
	if len(urlParts) >= 3 {
		consd = strings.Join(urlParts[:3], ".")
	}
	if consd == "console.s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinioAdmin) && remote.Status == string(transition.StartramServiceStatusOk) {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v console port to %v", patp, remote.Port))
		if remote.Port != local.WgConsolePort {
			local.WgConsolePort = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	}
	return false
}
