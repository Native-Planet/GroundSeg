package docker

import (
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

const (
	HermesContainerName             = "hermes"
	HermesDataVolumeName            = "hermes"
	HermesWorkspaceVolumeName       = "hermes_workspace"
	DefaultHermesImage              = "registry.hub.docker.com/nativeplanet/hermes-tlon:0.14.0-0.14.0"
	DefaultHermesModelProvider      = "openrouter"
	DefaultHermesModel              = "deepseek/deepseek-v4-flash"
	DefaultHermesVersion            = "0.14.0"
	DefaultHermesAgentRef           = "2ffa1c97c09317c1d066aa5708b8ad961a4ca589"
	DefaultHermesTlonAdapterVersion = "0.13.0"
	DefaultHermesTlonAdapterRef     = "b9180da6491d29933a98f6e4f1b1458ce61ca576"
	DefaultHermesDashboardHostPort  = 19119
	HermesDashboardContainerPort    = 9119
)

func HermesImageOrDefault(image string) string {
	if image = strings.TrimSpace(image); image != "" {
		return image
	}
	return DefaultHermesImage
}

func HermesModelProviderOrDefault(provider string) string {
	if provider = NormalizeHermesModelProvider(provider); provider != "" {
		return provider
	}
	return DefaultHermesModelProvider
}

func NormalizeHermesModelProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openrouter":
		return "openrouter"
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	default:
		return ""
	}
}

func HermesProviderAPIKeyEnv(provider string) string {
	switch NormalizeHermesModelProvider(provider) {
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	default:
		return ""
	}
}

func HermesModelOrDefault(model string) string {
	if model = strings.TrimSpace(model); model != "" {
		return model
	}
	return DefaultHermesModel
}

func HermesVersionOrDefault(version string) string {
	if version = strings.TrimSpace(version); version != "" {
		return version
	}
	return DefaultHermesVersion
}

func HermesAgentRefOrDefault(ref string) string {
	if ref = strings.TrimSpace(ref); ref != "" {
		return ref
	}
	return DefaultHermesAgentRef
}

func HermesTlonAdapterVersionOrDefault(version string) string {
	if version = strings.TrimSpace(version); version != "" {
		return version
	}
	return DefaultHermesTlonAdapterVersion
}

func HermesTlonAdapterRefOrDefault(ref string) string {
	if ref = strings.TrimSpace(ref); ref != "" {
		return ref
	}
	return DefaultHermesTlonAdapterRef
}

func NormalizeHermesShip(ship string) string {
	ship = strings.TrimSpace(ship)
	if ship == "" {
		return ""
	}
	if !strings.HasPrefix(ship, "~") {
		ship = "~" + ship
	}
	return ship
}

func LoadHermes() error {
	zap.L().Info("Loading Hermes")
	if err := config.LoadHermesConfig(); err != nil {
		return err
	}
	hermesConf := config.HermesConf()
	if !hermesConf.Enabled {
		stopDisabledHermes()
		return nil
	}
	if strings.TrimSpace(hermesConf.AccessCode) == "" {
		zap.L().Warn("Hermes is enabled but no access code is stored; restart Hermes from Profile")
		return nil
	}
	info, err := StartContainer(HermesContainerName, "hermes")
	if err != nil {
		return err
	}
	config.UpdateContainerState(HermesContainerName, info)
	return nil
}

func stopDisabledHermes() {
	existing, err := FindContainer(HermesContainerName)
	if err == nil && existing != nil && existing.State == "running" {
		if stopErr := StopContainerByName(HermesContainerName); stopErr != nil {
			zap.L().Warn(fmt.Sprintf("Unable to stop disabled Hermes container: %v", stopErr))
		}
	}
	if containerState, exists := config.GetContainerState()[HermesContainerName]; exists {
		containerState.DesiredStatus = "stopped"
		config.UpdateContainerState(HermesContainerName, containerState)
	}
}

func hermesContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	if containerName != HermesContainerName {
		return containerConfig, hostConfig, fmt.Errorf("invalid Hermes container name: %s", containerName)
	}
	if err := config.LoadHermesConfig(); err != nil {
		return containerConfig, hostConfig, err
	}
	hermesConf := config.HermesConf()
	if !hermesConf.Enabled {
		return containerConfig, hostConfig, fmt.Errorf("Hermes is not enabled")
	}
	owner := NormalizeHermesShip(hermesConf.Owner)
	if owner == "" {
		return containerConfig, hostConfig, fmt.Errorf("Hermes owner is not configured")
	}
	attachedShip := NormalizeHermesShip(hermesConf.Ship)
	if attachedShip == "" {
		return containerConfig, hostConfig, fmt.Errorf("Hermes ship is not configured")
	}
	accessCode := strings.TrimSpace(hermesConf.AccessCode)
	if accessCode == "" {
		return containerConfig, hostConfig, fmt.Errorf("Hermes access code is not configured")
	}
	if hermesConf.Port <= 0 {
		return containerConfig, hostConfig, fmt.Errorf("Hermes dashboard port is not configured")
	}
	patp := strings.TrimPrefix(attachedShip, "~")
	if err := config.LoadUrbitConfig(patp); err != nil {
		return containerConfig, hostConfig, err
	}
	shipConf := config.UrbitConf(patp)
	shipURL, err := hermesShipURL(shipConf)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	environment := []string{
		"HERMES_HOME=/opt/data",
		"HERMES_WORKSPACE=/workspace",
		"HERMES_CONTAINER_HOME=/workspace/home",
		"HOME=/workspace/home",
		"HERMES_DASHBOARD=1",
		"HERMES_DASHBOARD_HOST=0.0.0.0",
		fmt.Sprintf("HERMES_DASHBOARD_PORT=%d", HermesDashboardContainerPort),
		fmt.Sprintf("HERMES_MODEL_PROVIDER=%s", HermesModelProviderOrDefault(hermesConf.ModelProvider)),
		fmt.Sprintf("HERMES_MODEL=%s", HermesModelOrDefault(hermesConf.Model)),
		fmt.Sprintf("HERMES_AGENT_VERSION=%s", HermesVersionOrDefault(hermesConf.HermesVersion)),
		fmt.Sprintf("HERMES_AGENT_REF=%s", HermesAgentRefOrDefault(hermesConf.HermesAgentRef)),
		fmt.Sprintf("HERMES_TLON_ADAPTER_VERSION=%s", HermesTlonAdapterVersionOrDefault(hermesConf.TlonAdapterVersion)),
		fmt.Sprintf("HERMES_TLON_ADAPTER_REF=%s", HermesTlonAdapterRefOrDefault(hermesConf.TlonAdapterRef)),
		"TLON_TELEMETRY=false",
		"HERMES_TLON_TOOLSET=hermes-tlon",
		"TERMINAL_ENV=local",
		"TERMINAL_CWD=/workspace",
		"TERMINAL_LOCAL_PERSISTENT=true",
		"TERMINAL_TIMEOUT=180",
		"TERMINAL_MAX_FOREGROUND_TIMEOUT=900",
		"TLON_SKILL_PATH=/opt/tlon-apps/packages/tlon-skill/SKILL.md",
		"TLON_CLI=/usr/local/bin/tlon",
		"TLON_HOSTING=true",
		fmt.Sprintf("TLON_NODE_URL=%s", shipURL),
		fmt.Sprintf("TLON_NODE_ID=%s", attachedShip),
		fmt.Sprintf("TLON_ACCESS_CODE=%s", accessCode),
		fmt.Sprintf("TLON_OWNER_SHIP=%s", owner),
		fmt.Sprintf("TLON_HOME_CHANNEL=%s", owner),
		fmt.Sprintf("TLON_ALLOWED_USERS=%s", owner),
		fmt.Sprintf("TLON_DM_ALLOWLIST=%s", owner),
		fmt.Sprintf("TLON_DEFAULT_AUTHORIZED_SHIPS=%s", owner),
		"TLON_AUTO_DISCOVER=true",
		"TLON_AUTO_ACCEPT_DM_INVITES=true",
		"TLON_AUTO_ACCEPT_GROUP_INVITES=true",
		"TLON_ALLOW_ALL_USERS=false",
		"TLON_DM_POLL_ENABLED=true",
		"TLON_OWNER_LISTEN_ENABLED=true",
		fmt.Sprintf("URBIT_URL=%s", shipURL),
		fmt.Sprintf("URBIT_SHIP=%s", attachedShip),
		fmt.Sprintf("URBIT_CODE=%s", accessCode),
		fmt.Sprintf("TLON_URL=%s", shipURL),
		fmt.Sprintf("TLON_CODE=%s", accessCode),
		fmt.Sprintf("TLON_SHIP=%s", attachedShip),
		fmt.Sprintf("TLON_SHIP_URL=%s", shipURL),
		fmt.Sprintf("TLON_SHIP_NAME=%s", attachedShip),
		fmt.Sprintf("TLON_SHIP_CODE=%s", accessCode),
	}
	apiKeyEnv := HermesProviderAPIKeyEnv(hermesConf.ModelProvider)
	apiKey := strings.TrimSpace(hermesConf.ProviderAPIKey)
	if apiKeyEnv == "" {
		return containerConfig, hostConfig, fmt.Errorf("unsupported Hermes provider %q", hermesConf.ModelProvider)
	}
	if apiKey == "" {
		return containerConfig, hostConfig, fmt.Errorf("Hermes provider API key is not configured")
	}
	environment = append(environment, fmt.Sprintf("%s=%s", apiKeyEnv, apiKey))
	zap.L().Info(fmt.Sprintf("Configuring Hermes for %s via %s with owner %s", attachedShip, shipURL, owner))

	dashboardPort := nat.Port(fmt.Sprintf("%d/tcp", HermesDashboardContainerPort))
	containerConfig = container.Config{
		Image:        HermesImageOrDefault(hermesConf.Image),
		Env:          environment,
		Cmd:          []string{"gateway", "run", "--replace", "--accept-hooks"},
		ExposedPorts: nat.PortSet{dashboardPort: struct{}{}},
	}
	hostConfig = container.HostConfig{
		NetworkMode: "default",
		ExtraHosts:  []string{"host.docker.internal:host-gateway"},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: HermesDataVolumeName,
				Target: "/opt/data",
			},
			{
				Type:   mount.TypeVolume,
				Source: HermesWorkspaceVolumeName,
				Target: "/workspace",
			},
		},
		PortBindings: nat.PortMap{
			dashboardPort: []nat.PortBinding{
				{HostIP: hermesDashboardHostIP(), HostPort: fmt.Sprintf("%d", hermesConf.Port)},
			},
		},
	}
	return containerConfig, hostConfig, nil
}

func hermesShipURL(shipConf structs.UrbitDocker) (string, error) {
	if shipConf.Network == "wireguard" {
		if shipConf.WgHTTPPort <= 0 {
			return "", fmt.Errorf("wireguard HTTP port is not configured for Hermes")
		}
		return wireguardEndpoint(shipConf.WgHTTPPort)
	}
	if shipConf.HTTPPort <= 0 {
		return "", fmt.Errorf("HTTP port is not configured for Hermes")
	}
	return fmt.Sprintf("http://host.docker.internal:%d", shipConf.HTTPPort), nil
}

func hermesDashboardHostIP() string {
	if hostIP := strings.TrimSpace(os.Getenv("GROUNDSEG_HERMES_HOST_IP")); hostIP != "" {
		return hostIP
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to enumerate interfaces for Hermes dashboard binding: %v", err))
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !isCandidateLANInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil || !ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	zap.L().Warn("Unable to find a LAN interface for Hermes dashboard binding; falling back to localhost")
	return "127.0.0.1"
}

func isCandidateLANInterface(name string) bool {
	name = strings.ToLower(name)
	blockedPrefixes := []string{"br-", "docker", "veth", "wg", "tun", "tap"}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return !strings.Contains(name, "tailscale")
}

func ipFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch value := addr.(type) {
	case *net.IPNet:
		ip = value.IP
	case *net.IPAddr:
		ip = value.IP
	default:
		return nil
	}
	return ip.To4()
}
