ARG NODE_VERSION=22.22.2
FROM node:${NODE_VERSION}-bookworm-slim

ARG HERMES_AGENT_REPO=https://github.com/NousResearch/hermes-agent.git
ARG HERMES_AGENT_REF=9de9c25f620ff7f1ce0fd5457d596052d5159596
ARG HERMES_LCM_REPO=https://github.com/stephenschoettler/hermes-lcm.git
ARG HERMES_LCM_REF=34043036d12f69d0089c4630eb9acfb2d9424b4c
ARG TLON_APPS_REPO=https://github.com/tloncorp/tlon-apps.git
ARG TLON_APPS_REF=15fc26bd9d8a13345d54f4c7b3e00601a2e0996f
ARG PNPM_VERSION=11.9.0

LABEL org.opencontainers.image.title="NativePlanet Hermes Tlon"
LABEL org.opencontainers.image.description="Hermes Agent with the Tlon platform adapter and tlon CLI"
LABEL org.opencontainers.image.source="https://github.com/Native-Planet/GroundSeg"
LABEL org.nativeplanet.hermes.agent-ref="${HERMES_AGENT_REF}"
LABEL org.nativeplanet.hermes.lcm-ref="${HERMES_LCM_REF}"
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
ENV PATH="/opt/hermes-venv/bin:${PATH}"

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
    tmux \
    unzip \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g "pnpm@${PNPM_VERSION}"

RUN git init "${HERMES_AGENT_DIR}" \
  && cd "${HERMES_AGENT_DIR}" \
  && git fetch --depth=1 "${HERMES_AGENT_REPO}" "${HERMES_AGENT_REF}" \
  && git checkout --detach FETCH_HEAD

COPY patches /tmp/hermes-patches
RUN cd "${HERMES_AGENT_DIR}" \
  && for patch in /tmp/hermes-patches/*.patch; do git apply "$patch"; done \
  && rm -rf /tmp/hermes-patches

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

RUN git init /tmp/hermes-lcm \
  && cd /tmp/hermes-lcm \
  && git fetch --depth=1 "${HERMES_LCM_REPO}" "${HERMES_LCM_REF}" \
  && git checkout --detach FETCH_HEAD \
  && mkdir -p "${HERMES_AGENT_DIR}/plugins/context_engine/lcm" \
  && cp -a . "${HERMES_AGENT_DIR}/plugins/context_engine/lcm/" \
  && ln -sfn "${HERMES_AGENT_DIR}/plugins/context_engine/lcm" "${HERMES_AGENT_DIR}/hermes_lcm" \
  && rm -rf "${HERMES_AGENT_DIR}/plugins/context_engine/lcm/.git" /tmp/hermes-lcm

RUN git init "${TLON_APPS_DIR}" \
  && cd "${TLON_APPS_DIR}" \
  && git fetch --depth=1 "${TLON_APPS_REPO}" "${TLON_APPS_REF}" \
  && git checkout --detach FETCH_HEAD

RUN cd "${TLON_APPS_DIR}" \
  && pnpm install --filter @tloncorp/tlon-skill... --frozen-lockfile --ignore-scripts \
  && pnpm --filter @tloncorp/api build \
  && cd "${TLON_SKILL_DIR}" \
  && version="$(node -p "require('./package.json').version")" \
  && pnpm exec esbuild scripts/main.ts --bundle --platform=node --format=cjs --target=node22 --outfile=dist/tlon.cjs --define:__VERSION__=\"${version}\" \
  && pnpm store prune

RUN mkdir -p "${HERMES_HOME}" "${HERMES_WORKSPACE}" "${HERMES_CONTAINER_HOME}" \
  && ln -sf /usr/bin/python3 /usr/local/bin/python \
  && ln -sf "${HERMES_VENV}/bin/hermes" /usr/local/bin/hermes \
  && ln -sf "${HERMES_VENV}/bin/hermes-agent" /usr/local/bin/hermes-agent
VOLUME [ "/opt/data", "/workspace" ]
WORKDIR /workspace

COPY tlon /usr/local/bin/tlon
COPY entrypoint.sh /usr/local/bin/hermes-tlon-entrypoint
RUN chmod +x /usr/local/bin/tlon /usr/local/bin/hermes-tlon-entrypoint \
  && "${TLON_CLI}" --help >/dev/null

ENTRYPOINT ["/usr/bin/tini", "-g", "--", "/usr/local/bin/hermes-tlon-entrypoint"]
CMD ["gateway", "run", "--replace", "--accept-hooks"]
