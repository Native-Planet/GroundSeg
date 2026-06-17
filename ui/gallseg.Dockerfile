FROM node:23.6.0-bullseye-slim
ARG GS_PERIGEE_WASM_URL=https://files.native.computer/wasm/perigee.wasm
ARG GS_PERIGEE_WASM_EXEC_URL=https://files.native.computer/wasm/wasm_exec.js
ENV GS_URBIT_MODE true
ENV GS_PERIGEE_WASM_URL=$GS_PERIGEE_WASM_URL
ENV GS_PERIGEE_WASM_EXEC_URL=$GS_PERIGEE_WASM_EXEC_URL
RUN npm install -g npm@9
COPY ./src /webui/src
COPY ./static /webui/static
COPY .npmrc /webui/
COPY package-lock.json /webui/
COPY package.json /webui
COPY svelte.config.js /webui
COPY vite.config.js /webui
WORKDIR /webui
RUN npm install -g npm
RUN npm install
RUN npm run build
