package docker

import (
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"net"
	neturl "net/url"
	"os"
	"slices"
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
	HermesTlonSkillDir              = "/opt/data/tlon-skill"
	hermesConfigVersionLabel        = "nativeplanet.groundseg.hermes.config-version"
	hermesConfigVersion             = "2026-06-28-hermes-tmux-searxng"
	DefaultHermesImage              = "registry.hub.docker.com/nativeplanet/hermes-tlon:0.14.0-0.14.0"
	DefaultHermesModelProvider      = "openrouter"
	DefaultHermesModel              = "deepseek/deepseek-v4-flash"
	DefaultHermesVersion            = "0.14.0"
	DefaultHermesAgentRef           = "2ffa1c97c09317c1d066aa5708b8ad961a4ca589"
	DefaultHermesTlonAdapterVersion = "0.14.0"
	DefaultHermesTlonAdapterRef     = "33112008b1f3e83816dee61020dc5d4c57770c15"
	DefaultHermesDashboardHostPort  = 19119
	HermesDashboardContainerPort    = 9119
)

type hermesShipTarget struct {
	URL        string
	ExtraHosts []string
}

type hermesModelProvider struct {
	Name      string
	APIKeyEnv string
}

type hermesWebProvider struct {
	Name           string
	APIKeyEnv      string
	URLEnv         string
	AliasEnv       []string
	SearchBackend  string
	ExtractBackend string
}

var hermesModelProviders = []hermesModelProvider{
	{Name: "ai-gateway", APIKeyEnv: "AI_GATEWAY_API_KEY"},
	{Name: "alibaba", APIKeyEnv: "DASHSCOPE_API_KEY"},
	{Name: "alibaba-coding-plan", APIKeyEnv: "ALIBABA_CODING_PLAN_API_KEY"},
	{Name: "anthropic", APIKeyEnv: "ANTHROPIC_API_KEY"},
	{Name: "arcee", APIKeyEnv: "ARCEEAI_API_KEY"},
	{Name: "deepseek", APIKeyEnv: "DEEPSEEK_API_KEY"},
	{Name: "gmi", APIKeyEnv: "GMI_API_KEY"},
	{Name: "huggingface", APIKeyEnv: "HF_TOKEN"},
	{Name: "kilocode", APIKeyEnv: "KILOCODE_API_KEY"},
	{Name: "kimi-coding", APIKeyEnv: "KIMI_API_KEY"},
	{Name: "kimi-coding-cn", APIKeyEnv: "KIMI_CN_API_KEY"},
	{Name: "nous", APIKeyEnv: "NOUS_API_KEY"},
	{Name: "novita", APIKeyEnv: "NOVITA_API_KEY"},
	{Name: "nvidia", APIKeyEnv: "NVIDIA_API_KEY"},
	{Name: "ollama-cloud", APIKeyEnv: "OLLAMA_API_KEY"},
	{Name: "openai", APIKeyEnv: "OPENAI_API_KEY"},
	{Name: "opencode-go", APIKeyEnv: "OPENCODE_GO_API_KEY"},
	{Name: "opencode-zen", APIKeyEnv: "OPENCODE_ZEN_API_KEY"},
	{Name: "openrouter", APIKeyEnv: "OPENROUTER_API_KEY"},
	{Name: "stepfun", APIKeyEnv: "STEPFUN_API_KEY"},
	{Name: "xai", APIKeyEnv: "XAI_API_KEY"},
	{Name: "xiaomi", APIKeyEnv: "XIAOMI_API_KEY"},
	{Name: "zai", APIKeyEnv: "GLM_API_KEY"},
}

var hermesWebProviders = []hermesWebProvider{
	{Name: "brave-free", APIKeyEnv: "BRAVE_SEARCH_API_KEY", AliasEnv: []string{"BRAVE_API_KEY"}, SearchBackend: "brave-free"},
	{Name: "exa", APIKeyEnv: "EXA_API_KEY", SearchBackend: "exa", ExtractBackend: "exa"},
	{Name: "firecrawl", APIKeyEnv: "FIRECRAWL_API_KEY", SearchBackend: "firecrawl", ExtractBackend: "firecrawl"},
	{Name: "parallel", APIKeyEnv: "PARALLEL_API_KEY", SearchBackend: "parallel", ExtractBackend: "parallel"},
	{Name: "searxng", URLEnv: "SEARXNG_URL", SearchBackend: "searxng"},
	{Name: "tavily", APIKeyEnv: "TAVILY_API_KEY", SearchBackend: "tavily", ExtractBackend: "tavily"},
	{Name: "xai", APIKeyEnv: "XAI_API_KEY", SearchBackend: "xai"},
}

func HermesImageOrDefault(image string) string {
	if image = strings.TrimSpace(image); image != "" {
		return image
	}
	if image, err := HermesVersionServerImage(); err == nil && image != "" {
		return image
	}
	return DefaultHermesImage
}

func HermesVersionServerImage() (string, error) {
	info, err := GetLatestContainerInfo("hermes")
	if err != nil {
		return "", err
	}
	repo := strings.TrimSpace(info["repo"])
	tag := strings.TrimSpace(info["tag"])
	hash := strings.TrimSpace(info["hash"])
	if repo == "" || tag == "" {
		return "", fmt.Errorf("Hermes version-server image is missing repo or tag")
	}
	image := fmt.Sprintf("%s:%s", repo, tag)
	if hash != "" {
		image = fmt.Sprintf("%s@sha256:%s", image, hash)
	}
	return image, nil
}

func HermesUpdateAvailable(configuredImage string) bool {
	latestImage, err := HermesVersionServerImage()
	if err != nil || latestImage == "" {
		return false
	}
	currentImage := HermesImageOrDefault(configuredImage)
	return strings.TrimSpace(currentImage) != strings.TrimSpace(latestImage)
}

func HermesModelProviderOrDefault(provider string) string {
	if provider = NormalizeHermesModelProvider(provider); provider != "" {
		return provider
	}
	return DefaultHermesModelProvider
}

func NormalizeHermesModelProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, supported := range hermesModelProviders {
		if provider == supported.Name {
			return supported.Name
		}
	}
	return ""
}

func HermesProviderAPIKeyEnv(provider string) string {
	provider = NormalizeHermesModelProvider(provider)
	for _, supported := range hermesModelProviders {
		if provider == supported.Name {
			return supported.APIKeyEnv
		}
	}
	return ""
}

func HermesWebProviderOrEmpty(provider string) string {
	return NormalizeHermesWebProvider(provider)
}

func NormalizeHermesWebProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || provider == "off" || provider == "none" {
		return ""
	}
	for _, supported := range hermesWebProviders {
		if provider == supported.Name {
			return supported.Name
		}
	}
	return ""
}

func HermesWebProviderAPIKeyEnv(provider string) string {
	provider = NormalizeHermesWebProvider(provider)
	for _, supported := range hermesWebProviders {
		if provider == supported.Name {
			return supported.APIKeyEnv
		}
	}
	return ""
}

func HermesWebProviderURLEnv(provider string) string {
	provider = NormalizeHermesWebProvider(provider)
	for _, supported := range hermesWebProviders {
		if provider == supported.Name {
			return supported.URLEnv
		}
	}
	return ""
}

func HermesWebProviderConfig(provider string) (hermesWebProvider, bool) {
	provider = NormalizeHermesWebProvider(provider)
	for _, supported := range hermesWebProviders {
		if provider == supported.Name {
			return supported, true
		}
	}
	return hermesWebProvider{}, false
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
	shipTarget, err := hermesShipTargetForContainer(shipConf)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	shipURL := shipTarget.URL
	attachedShipBare := strings.TrimPrefix(attachedShip, "~")
	environment := []string{
		"HERMES_HOME=/opt/data",
		"HERMES_WORKSPACE=/workspace",
		"HERMES_WORKSPACE_DIR=/workspace",
		"HERMES_CONTAINER_HOME=/workspace/home",
		"HERMES_OPENROUTER_CACHE=false",
		"HERMES_TLON_ADAPTER_DIR=/opt/tlon-apps/packages/hermes-tlon-adapter",
		"HERMES_INTERACTIVE=true",
		"HERMES_GATEWAY_SESSION=true",
		"HERMES_EXEC_ASK=true",
		"LCM_DATABASE_PATH=/opt/data/lcm.db",
		"HOME=/workspace/home",
		"HERMES_DASHBOARD=1",
		"HERMES_DASHBOARD_HOST=0.0.0.0",
		fmt.Sprintf("HERMES_DASHBOARD_PORT=%d", HermesDashboardContainerPort),
		fmt.Sprintf("API_SERVER_ENABLED=%t", hermesConf.APIEnabled),
		"HERMES_ALLOW_CONFIG_WRITE=true",
		fmt.Sprintf("HERMES_INFERENCE_PROVIDER=%s", HermesModelProviderOrDefault(hermesConf.ModelProvider)),
		fmt.Sprintf("HERMES_MODEL_PROVIDER=%s", HermesModelProviderOrDefault(hermesConf.ModelProvider)),
		fmt.Sprintf("HERMES_MODEL=%s", HermesModelOrDefault(hermesConf.Model)),
		fmt.Sprintf("HERMES_AGENT_VERSION=%s", HermesVersionOrDefault(hermesConf.HermesVersion)),
		fmt.Sprintf("HERMES_AGENT_REF=%s", HermesAgentRefOrDefault(hermesConf.HermesAgentRef)),
		fmt.Sprintf("HERMES_TLON_ADAPTER_VERSION=%s", HermesTlonAdapterVersionOrDefault(hermesConf.TlonAdapterVersion)),
		fmt.Sprintf("HERMES_TLON_ADAPTER_REF=%s", HermesTlonAdapterRefOrDefault(hermesConf.TlonAdapterRef)),
		"TLON_TELEMETRY=false",
		"HERMES_TLON_TOOLSET=tlon",
		"HERMES_TLON_TOOLSETS=tlon,file,terminal,web,browser,skills,todo,cronjob,context_engine",
		"TERMINAL_ENV=local",
		"TERMINAL_CWD=/workspace",
		"TERMINAL_LOCAL_PERSISTENT=true",
		"TERMINAL_TIMEOUT=180",
		"TERMINAL_MAX_FOREGROUND_TIMEOUT=900",
		"TLON_SKILL_PATH=/opt/tlon-apps/packages/tlon-skill/SKILL.md",
		fmt.Sprintf("TLON_SKILL_DIR=%s", HermesTlonSkillDir),
		"TLON_CLI=/usr/local/bin/tlon",
		fmt.Sprintf("TLON_CONFIG_FILE=%s", hermesTlonShipConfigPath(attachedShipBare)),
		fmt.Sprintf("TLON_NODE_URL=%s", shipURL),
		fmt.Sprintf("TLON_NODE_ID=%s", attachedShip),
		fmt.Sprintf("TLON_ACCESS_CODE=%s", accessCode),
		fmt.Sprintf("TLON_OWNER=%s", owner),
		fmt.Sprintf("TLON_OWNER_SHIP=%s", owner),
		fmt.Sprintf("TLON_OWNER_URL=%s", shipURL),
		fmt.Sprintf("TLON_HOME_CHANNEL=%s", owner),
		fmt.Sprintf("TLON_ALLOWED_USERS=%s", owner),
		fmt.Sprintf("TLON_DM_ALLOWLIST=%s", owner),
		fmt.Sprintf("TLON_DEFAULT_AUTHORIZED_SHIPS=%s", owner),
		fmt.Sprintf("TLON_GROUP_INVITE_ALLOWLIST=%s", owner),
		"TLON_BOT_ALIASES=",
		"TLON_BOT_MENTIONS=",
		"TLON_CHANNELS=",
		"TLON_CHANNEL_RULES={}",
		"TLON_AUTO_DISCOVER=true",
		"TLON_AUTO_ACCEPT_DM_INVITES=true",
		"TLON_AUTO_ACCEPT_GROUP_INVITES=true",
		"TLON_ALLOW_ALL_USERS=false",
		"TLON_DM_POLL_ENABLED=true",
		"TLON_OWNER_LISTEN=true",
		"TLON_OWNER_LISTEN_ENABLED=true",
		"TLON_REQUIRE_MENTION=true",
		"TLON_MAX_CONSECUTIVE_BOT_RESPONSES=2",
		fmt.Sprintf("URBIT_URL=%s", shipURL),
		fmt.Sprintf("URBIT_SHIP=%s", attachedShip),
		fmt.Sprintf("URBIT_CODE=%s", accessCode),
		fmt.Sprintf("TLON_URL=%s", shipURL),
		fmt.Sprintf("TLON_CODE=%s", accessCode),
		fmt.Sprintf("TLON_SHIP=%s", attachedShipBare),
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
	if apiServerKey := strings.TrimSpace(hermesConf.APIKey); hermesConf.APIEnabled && apiServerKey != "" {
		environment = append(environment, fmt.Sprintf("API_SERVER_KEY=%s", apiServerKey))
	}
	if webProviderName := NormalizeHermesWebProvider(hermesConf.WebProvider); webProviderName != "" {
		webProvider, ok := HermesWebProviderConfig(webProviderName)
		if !ok {
			return containerConfig, hostConfig, fmt.Errorf("unsupported Hermes web provider %q", hermesConf.WebProvider)
		}
		environment = append(environment,
			fmt.Sprintf("HERMES_WEB_BACKEND=%s", webProvider.SearchBackend),
			fmt.Sprintf("HERMES_WEB_SEARCH_BACKEND=%s", webProvider.SearchBackend),
		)
		webAPIKey := strings.TrimSpace(hermesConf.WebAPIKey)
		if webProvider.APIKeyEnv != "" {
			if webAPIKey == "" {
				return containerConfig, hostConfig, fmt.Errorf("Hermes web API key is not configured")
			}
			environment = append(environment, fmt.Sprintf("%s=%s", webProvider.APIKeyEnv, webAPIKey))
		}
		if webProvider.URLEnv != "" {
			webURL, urlExtraHosts, err := hermesURLForContainer(hermesConf.WebURL)
			if err != nil {
				return containerConfig, hostConfig, err
			}
			if webURL == "" {
				return containerConfig, hostConfig, fmt.Errorf("Hermes web URL is not configured")
			}
			environment = append(environment, fmt.Sprintf("%s=%s", webProvider.URLEnv, webURL))
			shipTarget.ExtraHosts = appendUniqueStrings(shipTarget.ExtraHosts, urlExtraHosts...)
		}
		if webProvider.ExtractBackend != "" {
			environment = append(environment, fmt.Sprintf("HERMES_WEB_EXTRACT_BACKEND=%s", webProvider.ExtractBackend))
		}
		for _, alias := range webProvider.AliasEnv {
			environment = append(environment, fmt.Sprintf("%s=%s", alias, webAPIKey))
		}
	}
	zap.L().Info(fmt.Sprintf("Configuring Hermes for %s via %s with owner %s", attachedShip, shipURL, owner))

	dashboardPort := nat.Port(fmt.Sprintf("%d/tcp", HermesDashboardContainerPort))
	containerConfig = container.Config{
		Image:        HermesImageOrDefault(hermesConf.Image),
		Env:          environment,
		Cmd:          []string{"bash", "-lc", hermesGatewayCommand(hermesConf)},
		Labels:       map[string]string{hermesConfigVersionLabel: hermesConfigVersion},
		ExposedPorts: nat.PortSet{dashboardPort: struct{}{}},
	}
	hostConfig = container.HostConfig{
		NetworkMode: "default",
		ExtraHosts:  shipTarget.ExtraHosts,
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

func hermesGatewayCommand(hermesConf structs.HermesConfig) string {
	return fmt.Sprintf(
		`skill_dir="${TLON_SKILL_DIR:-/opt/data/tlon-skill}"
ship="${TLON_NODE_ID:-${TLON_SHIP_NAME:-${URBIT_SHIP:-${TLON_SHIP:-}}}}"
case "$ship" in
  "~"*) ;;
  "") ship="~ship" ;;
  *) ship="~$ship" ;;
esac
bare_ship="${ship#~}"
config_file="${TLON_CONFIG_FILE:-$skill_dir/ships/$bare_ship.json}"
mkdir -p "$(dirname "$config_file")" /opt/data /workspace
url="${TLON_NODE_URL:-${TLON_SHIP_URL:-${TLON_URL:-${URBIT_URL:-}}}}"
code="${TLON_ACCESS_CODE:-${TLON_SHIP_CODE:-${TLON_CODE:-${URBIT_CODE:-}}}}"
cat > "$config_file" <<EOF
{
  "url": "$url",
  "ship": "$ship",
  "code": "$code"
}
EOF
chmod 600 "$config_file"

managed_env_tmp="$(mktemp)"
{
  for name in \
    API_SERVER_ENABLED API_SERVER_KEY \
    AI_GATEWAY_API_KEY ALIBABA_CODING_PLAN_API_KEY ANTHROPIC_API_KEY ARCEEAI_API_KEY \
    BRAVE_API_KEY BRAVE_SEARCH_API_KEY DASHSCOPE_API_KEY DEEPSEEK_API_KEY \
    EXA_API_KEY FIRECRAWL_API_KEY GLM_API_KEY GMI_API_KEY GROQ_API_KEY \
    HF_TOKEN KILOCODE_API_KEY KIMI_API_KEY KIMI_CN_API_KEY KIMI_CODING_API_KEY \
    MISTRAL_API_KEY NOUS_API_KEY NOVITA_API_KEY NVIDIA_API_KEY OLLAMA_API_KEY \
    OPENCODE_GO_API_KEY OPENCODE_ZEN_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY \
    PARALLEL_API_KEY SEARXNG_URL STEPFUN_API_KEY TAVILY_API_KEY XAI_API_KEY XIAOMI_API_KEY ZAI_API_KEY Z_AI_API_KEY \
    HERMES_CONTAINER_HOME HERMES_DASHBOARD HERMES_DASHBOARD_HOST HERMES_DASHBOARD_PORT \
    HERMES_ALLOW_CONFIG_WRITE HERMES_EXEC_ASK HERMES_GATEWAY_SESSION HERMES_HOME HERMES_INFERENCE_PROVIDER \
    HERMES_INTERACTIVE HERMES_MODEL HERMES_MODEL_PROVIDER HERMES_OPENROUTER_CACHE \
    HERMES_TLON_ADAPTER_DIR HERMES_TLON_TOOLSET HERMES_TLON_TOOLSETS HERMES_WEB_BACKEND \
    HERMES_WEB_EXTRACT_BACKEND HERMES_WEB_SEARCH_BACKEND HERMES_WORKSPACE \
    HERMES_WORKSPACE_DIR HOME LCM_DATABASE_PATH TERMINAL_CWD TERMINAL_ENV TERMINAL_LOCAL_PERSISTENT \
    TERMINAL_MAX_FOREGROUND_TIMEOUT TERMINAL_TIMEOUT TLON_ACCESS_CODE TLON_ALLOWED_USERS \
    TLON_ALLOW_ALL_USERS TLON_AUTO_ACCEPT_DM_INVITES TLON_AUTO_ACCEPT_GROUP_INVITES \
    TLON_AUTO_DISCOVER TLON_BOT_ALIASES TLON_BOT_MENTIONS TLON_CHANNELS TLON_CHANNEL_RULES \
	TLON_CLI TLON_CODE TLON_DEFAULT_AUTHORIZED_SHIPS TLON_DM_ALLOWLIST \
	TLON_DM_POLL_ENABLED TLON_GROUP_INVITE_ALLOWLIST TLON_HOME_CHANNEL \
	TLON_MAX_CONSECUTIVE_BOT_RESPONSES TLON_NODE_ID TLON_NODE_URL TLON_OWNER \
    TLON_OWNER_LISTEN TLON_OWNER_LISTEN_ENABLED TLON_OWNER_SHIP TLON_OWNER_URL \
    TLON_REQUIRE_MENTION TLON_SHIP TLON_SHIP_CODE TLON_SHIP_NAME TLON_SHIP_URL \
    TLON_SKILL_PATH TLON_TELEMETRY TLON_URL URBIT_CODE URBIT_SHIP \
    URBIT_URL
  do
    eval "value=\${$name-}"
    printf '%%s=%%s\n' "$name" "$value"
  done
  printf 'TLON_CONFIG_FILE=%%s\n' "$config_file"
  printf 'TLON_SKILL_DIR=%%s\n' "$skill_dir"
} > "$managed_env_tmp"
if [ -f /opt/data/.env ]; then
  awk -F= '
    NR == FNR {
      if ($0 ~ /^[A-Za-z_][A-Za-z0-9_]*=/) managed[$1] = 1
      next
    }
    /^[[:space:]]*(#|$)/ { print; next }
    {
      line = $0
      sub(/^[[:space:]]*export[[:space:]]+/, "", line)
      if (line ~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
        key = line
        sub(/=.*/, "", key)
        if (managed[key]) next
      }
      print
    }
  ' "$managed_env_tmp" /opt/data/.env > /opt/data/.env.next
else
  : > /opt/data/.env.next
fi
cat "$managed_env_tmp" >> /opt/data/.env.next
mv /opt/data/.env.next /opt/data/.env
rm -f "$managed_env_tmp"
chmod 600 /opt/data/.env
cp /opt/data/.env /workspace/.env
chmod 600 /workspace/.env
echo "Hermes Tlon runtime files: env=/opt/data/.env workspace_env=/workspace/.env config=$config_file"
echo "Hermes Tlon CLI: ${TLON_CLI:-tlon} ($(command -v "${TLON_CLI:-tlon}" || true))"
if ! "${TLON_CLI:-tlon}" --help >/dev/null 2>&1; then
  echo "ERROR: tlon CLI failed its startup smoke check" >&2
  "${TLON_CLI:-tlon}" --help >/dev/null
  exit 1
fi

if command -v tmux >/dev/null 2>&1; then
  log_file="/opt/data/logs/gateway.log"
  exit_file="/opt/data/gateway.exit"
  mkdir -p "$(dirname "$log_file")"
  : > "$log_file"
  rm -f "$exit_file"
  tmux kill-session -t hermes >/dev/null 2>&1 || true
  tmux new-session -d -s hermes -n gateway "bash -lc 'set -o pipefail; hermes gateway run --replace --accept-hooks 2>&1 | tee -a /opt/data/logs/gateway.log; code=\${PIPESTATUS[0]}; echo \"\$code\" > /opt/data/gateway.exit; tmux wait-for -S hermes-gateway-exit; exit \"\$code\"'"
  tmux new-window -d -t hermes -n shell "bash -l"
  tmux select-window -t hermes:shell

  cleanup() {
    tmux send-keys -t hermes:gateway C-c >/dev/null 2>&1 || true
    sleep 2
    tmux kill-session -t hermes >/dev/null 2>&1 || true
  }
  trap cleanup INT TERM

  tail -n +1 -F "$log_file" &
  tail_pid="$!"
  tmux wait-for hermes-gateway-exit || true
  kill "$tail_pid" >/dev/null 2>&1 || true
  wait "$tail_pid" >/dev/null 2>&1 || true
  code="1"
  if [ -f "$exit_file" ]; then
    code="$(cat "$exit_file")"
  fi
  exit "$code"
fi
exec hermes gateway run --replace --accept-hooks`,
	)
}

func hermesTlonShipConfigPath(attachedShipBare string) string {
	return fmt.Sprintf("%s/ships/%s.json", HermesTlonSkillDir, strings.TrimPrefix(attachedShipBare, "~"))
}

func hermesShipTargetForContainer(shipConf structs.UrbitDocker) (hermesShipTarget, error) {
	if shipConf.Network == "wireguard" {
		remoteURL := UrbitRemoteWebURL(shipConf)
		if remoteURL == "" {
			return hermesShipTarget{}, fmt.Errorf("remote URL is not configured for Hermes")
		}

		return hermesShipTarget{
			URL: remoteURL,
		}, nil
	}
	if shipConf.HTTPPort <= 0 {
		return hermesShipTarget{}, fmt.Errorf("HTTP port is not configured for Hermes")
	}
	return hermesShipTarget{
		URL:        fmt.Sprintf("http://host.docker.internal:%d", shipConf.HTTPPort),
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}, nil
}

func UrbitWebURL(localHost string, shipConf structs.UrbitDocker) string {
	if remoteURL := UrbitRemoteWebURL(shipConf); remoteURL != "" {
		return remoteURL
	}
	localHost = strings.TrimSpace(localHost)
	if localHost == "" || shipConf.HTTPPort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", localHost, shipConf.HTTPPort)
}

func UrbitRemoteWebURL(shipConf structs.UrbitDocker) string {
	if shipConf.Network != "wireguard" {
		return ""
	}
	remoteURL := strings.TrimSpace(shipConf.WgURL)
	customURL := strings.TrimSpace(shipConf.CustomUrbitWeb)
	if strings.EqualFold(customURL, "null") {
		customURL = ""
	}
	if shipConf.ShowUrbitWeb == "custom" && customURL != "" {
		remoteURL = customURL
	}
	return normalizeHermesURL(remoteURL)
}

func normalizeHermesURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	return "https://" + rawURL
}

func hermesURLForContainer(rawURL string) (string, []string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil, nil
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	parsed, err := neturl.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", nil, fmt.Errorf("invalid Hermes web URL %q", rawURL)
	}
	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		port := parsed.Port()
		if port != "" {
			parsed.Host = net.JoinHostPort("host.docker.internal", port)
		} else {
			parsed.Host = "host.docker.internal"
		}
		return strings.TrimRight(parsed.String(), "/"), []string{"host.docker.internal:host-gateway"}, nil
	}
	return strings.TrimRight(parsed.String(), "/"), nil, nil
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		if addition == "" {
			continue
		}
		exists := slices.Contains(values, addition)
		if !exists {
			values = append(values, addition)
		}
	}
	return values
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
