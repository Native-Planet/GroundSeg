ARG NODE_VERSION=22.22.2
FROM node:${NODE_VERSION}-bookworm-slim

ARG HERMES_AGENT_REPO=https://github.com/NousResearch/hermes-agent.git
ARG HERMES_AGENT_REF=2ffa1c97c09317c1d066aa5708b8ad961a4ca589
ARG TLON_APPS_REPO=https://github.com/tloncorp/tlon-apps.git
ARG TLON_APPS_REF=b9180da6491d29933a98f6e4f1b1458ce61ca576
ARG PNPM_VERSION=9.15.9

LABEL org.opencontainers.image.title="NativePlanet Hermes Tlon"
LABEL org.opencontainers.image.description="Hermes Agent with the Tlon platform adapter and tlon CLI"
LABEL org.opencontainers.image.source="https://github.com/Native-Planet/GroundSeg"
LABEL org.nativeplanet.hermes.agent-ref="${HERMES_AGENT_REF}"
LABEL org.nativeplanet.hermes.tlon-apps-ref="${TLON_APPS_REF}"

ENV HERMES_AGENT_DIR=/opt/hermes-agent
ENV HERMES_VENV=/opt/hermes-venv
ENV TLON_APPS_DIR=/opt/tlon-apps
ENV TLON_ADAPTER_DIR=/opt/tlon-apps/packages/hermes-tlon-adapter
ENV TLON_SKILL_DIR=/opt/tlon-apps/packages/tlon-skill
ENV TLON_CLI=/usr/local/bin/tlon
ENV TLON_SKILL_PATH=/opt/tlon-apps/packages/tlon-skill/SKILL.md
ENV HERMES_HOME=/opt/data
ENV HERMES_WEB_DIST=/opt/hermes-agent/hermes_cli/web_dist
ENV HERMES_DASHBOARD=1
ENV HERMES_DASHBOARD_HOST=0.0.0.0
ENV HERMES_DASHBOARD_PORT=9119
ENV HERMES_MODEL_PROVIDER=openrouter
ENV HERMES_MODEL=deepseek/deepseek-v4-flash
ENV TLON_TELEMETRY=false
ENV TERMINAL_CWD=/opt/data
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
    unzip \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g "pnpm@${PNPM_VERSION}" \
  && curl -fsSL https://bun.sh/install | bash

RUN git clone --filter=blob:none "${HERMES_AGENT_REPO}" "${HERMES_AGENT_DIR}" \
  && cd "${HERMES_AGENT_DIR}" \
  && git checkout "${HERMES_AGENT_REF}"

RUN cd "${HERMES_AGENT_DIR}" \
  && npm install --prefer-offline --no-audit \
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
  && cp "${TLON_SKILL_DIR}/dist/tlon-run" "${TLON_CLI}" \
  && chmod +x "${TLON_CLI}" \
  && "${TLON_CLI}" --version \
  && pnpm store prune

RUN mkdir -p "${HERMES_HOME}"
VOLUME [ "/opt/data" ]

COPY entrypoint.sh /usr/local/bin/hermes-tlon-entrypoint
RUN chmod +x /usr/local/bin/hermes-tlon-entrypoint

ENTRYPOINT ["/usr/local/bin/hermes-tlon-entrypoint"]
CMD ["gateway", "run", "--replace", "--accept-hooks"]
