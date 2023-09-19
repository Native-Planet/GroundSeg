FROM node:18.17.0-buster-slim
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