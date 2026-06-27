ARG NODE_VERSION=22.22.2
FROM node:${NODE_VERSION}-bookworm-slim

ARG HERMES_AGENT_REPO=https://github.com/NousResearch/hermes-agent.git
ARG HERMES_AGENT_REF=2ffa1c97c09317c1d066aa5708b8ad961a4ca589
ARG TLON_APPS_REPO=https://github.com/tloncorp/tlon-apps.git
ARG TLON_APPS_REF=b9180da6491d29933a98f6e4f1b1458ce61ca576
ARG TLON_SKILL_CLI_VERSION=0.4.0
ARG PNPM_VERSION=9.15.9
ARG TARGETARCH

LABEL org.opencontainers.image.title="NativePlanet Hermes Tlon"
LABEL org.opencontainers.image.description="Hermes Agent with the Tlon platform adapter and tlon CLI"
LABEL org.opencontainers.image.source="https://github.com/Native-Planet/GroundSeg"
LABEL org.nativeplanet.hermes.agent-ref="${HERMES_AGENT_REF}"
LABEL org.nativeplanet.hermes.tlon-apps-ref="${TLON_APPS_REF}"

ENV HERMES_AGENT_DIR=/opt/hermes-agent
ENV HERMES_VENV=/opt/hermes-venv
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/hermes/.playwright
ENV TLON_APPS_DIR=/opt/tlon-apps
ENV TLON_ADAPTER_DIR=/opt/tlon-apps/packages/hermes-tlon-adapter
ENV TLON_SKILL_DIR=/opt/tlon-apps/packages/tlon-skill
ENV TLON_CLI=/usr/local/bin/tlon
ENV TLON_SKILL_PATH=/opt/tlon-apps/packages/tlon-skill/SKILL.md
ENV HERMES_HOME=/opt/data
ENV HERMES_WORKSPACE=/workspace
ENV HERMES_CONTAINER_HOME=/workspace/home
ENV HERMES_WEB_DIST=/opt/hermes-agent/hermes_cli/web_dist
ENV HERMES_DASHBOARD=1
ENV HERMES_DASHBOARD_HOST=0.0.0.0
ENV HERMES_DASHBOARD_PORT=9119
ENV HERMES_MODEL_PROVIDER=openrouter
ENV HERMES_MODEL=deepseek/deepseek-v4-flash
ENV TLON_TELEMETRY=false
ENV TERMINAL_ENV=local
ENV TERMINAL_CWD=/workspace
ENV TERMINAL_LOCAL_PERSISTENT=true
ENV TERMINAL_TIMEOUT=180
ENV TERMINAL_MAX_FOREGROUND_TIMEOUT=900
ENV PATH="/opt/hermes-venv/bin:/root/.bun/bin:${PATH}"

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    build-essential \
    ca-certificates \
    curl \
    ffmpeg \
    git \
    libffi-dev \
    openssh-client \
    procps \
    python3 \
    python3-dev \
    python3-pip \
    python3-venv \
    ripgrep \
    tini \
    unzip \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g "pnpm@${PNPM_VERSION}" \
  && curl -fsSL https://bun.sh/install | bash

RUN git clone --filter=blob:none "${HERMES_AGENT_REPO}" "${HERMES_AGENT_DIR}" \
  && cd "${HERMES_AGENT_DIR}" \
  && git checkout "${HERMES_AGENT_REF}"

RUN cd "${HERMES_AGENT_DIR}" \
  && npm install --prefer-offline --no-audit \
  && npx playwright install --with-deps chromium --only-shell \
  && cd web \
  && npm install --prefer-offline --no-audit \
  && npm run build \
  && cd ../ui-tui \
  && npm install --prefer-offline --no-audit \
  && npm run build \
  && npm cache clean --force

RUN python3 -m venv "${HERMES_VENV}" \
  && "${HERMES_VENV}/bin/python" -m pip install --upgrade pip setuptools wheel \
  && "${HERMES_VENV}/bin/python" -m pip install -e "${HERMES_AGENT_DIR}[cron,cli,pty,mcp,acp,web]" \
  && "${HERMES_VENV}/bin/python" -m pip install "aiohttp>=3.9"

RUN git clone --filter=blob:none "${TLON_APPS_REPO}" "${TLON_APPS_DIR}" \
  && cd "${TLON_APPS_DIR}" \
  && git checkout "${TLON_APPS_REF}"

RUN cd "${TLON_APPS_DIR}" \
  && pnpm install --filter @tloncorp/tlon-skill... --frozen-lockfile --ignore-scripts \
  && pnpm --filter @tloncorp/api build \
  && pnpm --filter @tloncorp/tlon-skill build \
  && pnpm store prune

RUN set -eux; \
  arch="${TARGETARCH:-$(dpkg --print-architecture)}"; \
  case "$arch" in \
    amd64|x86_64) tlon_pkg_arch="linux-x64" ;; \
    arm64|aarch64) tlon_pkg_arch="linux-arm64" ;; \
    *) echo "Unsupported tlon CLI architecture: $arch" >&2; exit 1 ;; \
  esac; \
  tmp="$(mktemp -d)"; \
  curl -fsSL "https://registry.npmjs.org/@tloncorp/tlon-skill-${tlon_pkg_arch}/-/tlon-skill-${tlon_pkg_arch}-${TLON_SKILL_CLI_VERSION}.tgz" \
    | tar -xz -C "$tmp"; \
  install -m 0755 "$tmp/package/tlon" "${TLON_CLI}"; \
  rm -rf "$tmp"

RUN mkdir -p "${HERMES_HOME}" "${HERMES_WORKSPACE}" "${HERMES_CONTAINER_HOME}" \
  && ln -sf /root/.bun/bin/bun /usr/local/bin/bun \
  && ln -sf /root/.bun/bin/bun /usr/local/bin/bunx \
  && ln -sf /usr/bin/python3 /usr/local/bin/python \
  && ln -sf "${HERMES_VENV}/bin/hermes" /usr/local/bin/hermes \
  && ln -sf "${HERMES_VENV}/bin/hermes-agent" /usr/local/bin/hermes-agent
VOLUME [ "/opt/data", "/workspace" ]
WORKDIR /workspace

COPY entrypoint.sh /usr/local/bin/hermes-tlon-entrypoint
RUN chmod +x /usr/local/bin/hermes-tlon-entrypoint

ENTRYPOINT ["/usr/bin/tini", "-g", "--", "/usr/local/bin/hermes-tlon-entrypoint"]
CMD ["gateway", "run", "--replace", "--accept-hooks"]
