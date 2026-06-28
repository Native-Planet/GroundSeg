#!/usr/bin/env bash
set -euo pipefail

HERMES_HOME="${HERMES_HOME:-/opt/data}"
HERMES_WORKSPACE="${HERMES_WORKSPACE:-/workspace}"
HERMES_CONTAINER_HOME="${HERMES_CONTAINER_HOME:-$HERMES_WORKSPACE/home}"
HOME="$HERMES_CONTAINER_HOME"
HERMES_AGENT_DIR="${HERMES_AGENT_DIR:-/opt/hermes-agent}"
TLON_ADAPTER_DIR="${TLON_ADAPTER_DIR:-/opt/tlon-apps/packages/hermes-tlon-adapter}"
TLON_SKILL_DIR="${TLON_SKILL_DIR:-/opt/tlon-apps/packages/tlon-skill}"
TLON_CLI="${TLON_CLI:-/usr/local/bin/tlon}"
TERMINAL_ENV="${TERMINAL_ENV:-local}"
TERMINAL_CWD="${TERMINAL_CWD:-$HERMES_WORKSPACE}"
TERMINAL_LOCAL_PERSISTENT="${TERMINAL_LOCAL_PERSISTENT:-true}"
TERMINAL_TIMEOUT="${TERMINAL_TIMEOUT:-180}"
TERMINAL_MAX_FOREGROUND_TIMEOUT="${TERMINAL_MAX_FOREGROUND_TIMEOUT:-900}"
HERMES_TLON_TOOLSET="${HERMES_TLON_TOOLSET:-hermes-tlon}"

export HERMES_HOME HERMES_WORKSPACE HERMES_CONTAINER_HOME HOME
export HERMES_AGENT_DIR TLON_ADAPTER_DIR TLON_SKILL_DIR TLON_CLI
export TERMINAL_ENV TERMINAL_CWD TERMINAL_LOCAL_PERSISTENT TERMINAL_TIMEOUT TERMINAL_MAX_FOREGROUND_TIMEOUT
export HERMES_TLON_TOOLSET

if [ -z "${BRAVE_SEARCH_API_KEY:-}" ] && [ -n "${BRAVE_API_KEY:-}" ]; then
  export BRAVE_SEARCH_API_KEY="$BRAVE_API_KEY"
fi

if [ -z "${TLON_HOME_CHANNEL:-}" ] && [ -n "${TLON_OWNER_SHIP:-}" ]; then
  export TLON_HOME_CHANNEL="$TLON_OWNER_SHIP"
fi

if [ ! -f "$TLON_ADAPTER_DIR/plugin.yaml" ]; then
  echo "ERROR: Tlon Hermes adapter is missing at $TLON_ADAPTER_DIR" >&2
  exit 1
fi

if [ ! -x "$TLON_CLI" ]; then
  echo "ERROR: tlon CLI is missing or not executable at $TLON_CLI" >&2
  exit 1
fi

if ! "$TLON_CLI" --help >/dev/null 2>&1; then
  echo "ERROR: tlon CLI failed its startup smoke check" >&2
  "$TLON_CLI" --help >/dev/null
  exit 1
fi

mkdir -p "$HERMES_HOME/plugins/platforms" "$HERMES_HOME/logs" "$HERMES_HOME/memories" "$HERMES_WORKSPACE" "$HERMES_CONTAINER_HOME"
ln -sfn "$TLON_ADAPTER_DIR" "$HERMES_HOME/plugins/platforms/tlon"

python3 - <<'PY'
import os
import re
from pathlib import Path

import yaml

home = Path(os.environ["HERMES_HOME"])
workspace = Path(os.environ.get("HERMES_WORKSPACE") or "/workspace")
adapter_dir = Path(os.environ["TLON_ADAPTER_DIR"])
prompts_dir = adapter_dir / "prompts"
prompts_root = prompts_dir.resolve()
include_re = re.compile(r"(?m)^\{\{include:([^}]+)\}\}\s*$")


def env_any(names, default):
    for name in names:
        value = (os.environ.get(name) or "").strip()
        if value:
            return value
    return default


def env_int(name, default):
    try:
        return int(os.environ.get(name) or default)
    except (TypeError, ValueError):
        return default


values = {
    "TLON_NODE_ID": env_any(["TLON_NODE_ID", "TLON_SHIP", "URBIT_SHIP"], "the configured bot node"),
    "TLON_OWNER_SHIP": env_any(["TLON_OWNER_SHIP"], "the configured owner ship"),
    "TLON_NODE_URL": env_any(["TLON_NODE_URL", "TLON_SHIP_URL", "TLON_URL", "URBIT_URL"], "the configured Tlon node URL"),
}


def checked_prompt_path(rel):
    path = (prompts_dir / rel).resolve()
    if path != prompts_root and prompts_root not in path.parents:
        raise ValueError(f"Prompt include escapes prompts directory: {rel}")
    return path


def render_prompt(rel, stack=()):
    if rel in stack:
        raise ValueError(f"Prompt include cycle: {' -> '.join((*stack, rel))}")
    path = checked_prompt_path(rel)
    text = path.read_text(encoding="utf-8")

    def include(match):
        return render_prompt(match.group(1).strip(), (*stack, rel)).rstrip()

    text = include_re.sub(include, text)
    for key, value in values.items():
        text = text.replace("{{" + key + "}}", value)
    return text.strip() + "\n"


def upsert_managed_block(target, rel, *, replace_default_soul=False, memory_file=False):
    rendered = render_prompt(rel).rstrip()
    start = f"<!-- BEGIN tlon-managed:{rel} -->"
    end = f"<!-- END tlon-managed:{rel} -->"
    block = f"{start}\n{rendered}\n{end}\n"
    target.parent.mkdir(parents=True, exist_ok=True)
    current = target.read_text(encoding="utf-8") if target.exists() else ""
    default_soul = "You are Hermes Agent, an intelligent AI assistant created by Nous Research."

    if start in current and end in current:
        pattern = re.compile(re.escape(start) + r".*?" + re.escape(end) + r"\n?", re.S)
        updated = pattern.sub(block, current)
    elif replace_default_soul and current.strip().startswith(default_soul):
        updated = block
    elif current.strip():
        separator = "\n---\n" if memory_file else "\n\n"
        updated = current.rstrip() + separator + block
    else:
        updated = block
    target.write_text(updated, encoding="utf-8")


upsert_managed_block(home / "SOUL.md", "hermes/SOUL.md", replace_default_soul=True)
upsert_managed_block(home / ".hermes.md", "hermes/.hermes.md")
upsert_managed_block(home / "memories" / "USER.md", "hermes/USER.md", memory_file=True)

config_path = home / "config.yaml"
config = yaml.safe_load(config_path.read_text()) if config_path.exists() else {}
if not isinstance(config, dict):
    config = {}

plugins = config.setdefault("plugins", {})
enabled = plugins.setdefault("enabled", [])
if not isinstance(enabled, list):
    enabled = []
    plugins["enabled"] = enabled
if "platforms/tlon" not in enabled:
    enabled.append("platforms/tlon")

gateway = config.setdefault("gateway", {})
gateway_platforms = gateway.setdefault("platforms", {})
gateway_tlon = gateway_platforms.setdefault("tlon", {})
gateway_tlon["enabled"] = True

platforms = config.setdefault("platforms", {})
tlon = platforms.setdefault("tlon", {})
tlon["enabled"] = True

terminal = config.get("terminal")
if not isinstance(terminal, dict):
    terminal = {}
terminal["backend"] = os.environ.get("TERMINAL_ENV") or "local"
terminal["cwd"] = os.environ.get("TERMINAL_CWD") or str(workspace)
terminal["timeout"] = env_int("TERMINAL_TIMEOUT", 180)
terminal["persistent_shell"] = str(os.environ.get("TERMINAL_LOCAL_PERSISTENT") or "true").lower() in {"true", "1", "yes"}
config["terminal"] = terminal

toolsets_raw = (
    os.environ.get("HERMES_TLON_TOOLSETS")
    or os.environ.get("HERMES_TLON_TOOLSET")
    or "hermes-tlon"
)
toolsets_selected = []
for item in re.split(r"[,:\s]+", toolsets_raw):
    item = item.strip()
    if item and item not in toolsets_selected:
        toolsets_selected.append(item)
if toolsets_selected:
    toolsets = config.get("toolsets")
    if not isinstance(toolsets, list):
        toolsets = []
    for toolset in toolsets_selected:
        if toolset not in toolsets:
            toolsets.append(toolset)
    config["toolsets"] = toolsets

    platform_toolsets = config.get("platform_toolsets")
    if not isinstance(platform_toolsets, dict):
        platform_toolsets = {}
    tlon_toolsets = platform_toolsets.get("tlon")
    if not isinstance(tlon_toolsets, list) or not tlon_toolsets:
        tlon_toolsets = list(toolsets_selected)
    else:
        current = {str(item).strip() for item in tlon_toolsets if str(item).strip()}
        if current <= {"tlon", "hermes-tlon"}:
            tlon_toolsets = list(toolsets_selected)
    platform_toolsets["tlon"] = tlon_toolsets
    config["platform_toolsets"] = platform_toolsets

home_channel = (
    os.environ.get("TLON_HOME_CHANNEL")
    or os.environ.get("TLON_OWNER_SHIP")
    or os.environ.get("TLON_GATEWAY_STATUS_OWNER")
    or ""
).strip()
if home_channel:
    home_channel_config = {
        "platform": "tlon",
        "chat_id": home_channel,
        "name": home_channel,
    }
    tlon["home_channel"] = home_channel_config
    gateway_tlon["home_channel"] = home_channel_config

provider = (os.environ.get("HERMES_MODEL_PROVIDER") or os.environ.get("HERMES_PROVIDER") or "").strip()
model = (os.environ.get("HERMES_MODEL") or os.environ.get("MODEL") or "").strip()
if provider or model:
    model_config = config.get("model")
    if not isinstance(model_config, dict):
        model_config = {}
    if provider:
        model_config["provider"] = provider
    if model:
        model_config["default"] = model
    config["model"] = model_config

web_backend = (os.environ.get("HERMES_WEB_BACKEND") or "").strip()
web_search_backend = (os.environ.get("HERMES_WEB_SEARCH_BACKEND") or web_backend).strip()
if not web_search_backend:
    if (os.environ.get("BRAVE_SEARCH_API_KEY") or "").strip():
        web_search_backend = "brave-free"
    elif (os.environ.get("EXA_API_KEY") or "").strip():
        web_search_backend = "exa"
    elif (os.environ.get("FIRECRAWL_API_KEY") or "").strip():
        web_search_backend = "firecrawl"
    elif (os.environ.get("PARALLEL_API_KEY") or "").strip():
        web_search_backend = "parallel"
    elif (os.environ.get("SEARXNG_URL") or "").strip():
        web_search_backend = "searxng"
    elif (os.environ.get("TAVILY_API_KEY") or "").strip():
        web_search_backend = "tavily"
    elif (os.environ.get("XAI_API_KEY") or "").strip():
        web_search_backend = "xai"

web_extract_backend = (os.environ.get("HERMES_WEB_EXTRACT_BACKEND") or "").strip()
if not web_extract_backend:
    if (os.environ.get("EXA_API_KEY") or "").strip():
        web_extract_backend = "exa"
    elif (os.environ.get("FIRECRAWL_API_KEY") or "").strip():
        web_extract_backend = "firecrawl"
    elif (os.environ.get("PARALLEL_API_KEY") or "").strip():
        web_extract_backend = "parallel"
    elif (os.environ.get("TAVILY_API_KEY") or "").strip():
        web_extract_backend = "tavily"
if web_search_backend or web_extract_backend:
    web_config = config.get("web")
    if not isinstance(web_config, dict):
        web_config = {}
    if web_search_backend:
        web_config["search_backend"] = web_search_backend
    if web_extract_backend:
        web_config["extract_backend"] = web_extract_backend
    config["web"] = web_config

config_path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
PY

if [ -d "$HERMES_AGENT_DIR/skills" ] && [ -f "$HERMES_AGENT_DIR/tools/skills_sync.py" ]; then
  python3 "$HERMES_AGENT_DIR/tools/skills_sync.py" || true
fi

case "${HERMES_DASHBOARD:-}" in
  1|true|TRUE|True|yes|YES|Yes)
    dash_host="${HERMES_DASHBOARD_HOST:-0.0.0.0}"
    dash_port="${HERMES_DASHBOARD_PORT:-9119}"
    dash_args=(--host "$dash_host" --port "$dash_port" --no-open)
    if [ "$dash_host" != "127.0.0.1" ] && [ "$dash_host" != "localhost" ]; then
      dash_args+=(--insecure)
    fi
    (
      stdbuf -oL -eL hermes dashboard "${dash_args[@]}" 2>&1 \
        | sed -u 's/^/[dashboard] /'
    ) &
    ;;
esac

if [ $# -gt 0 ] && command -v "$1" >/dev/null 2>&1; then
  exec "$@"
fi

exec hermes "$@"
