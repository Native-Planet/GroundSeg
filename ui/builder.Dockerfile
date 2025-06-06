FROM node:23.6.0-bullseye-slim
ARG GS_VERSION
ENV GS_VERSION=$GS_VERSION
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