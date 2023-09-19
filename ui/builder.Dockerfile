FROM node:18.17.0-buster-slim
RUN npm install -g npm@9
COPY ./src /webui
COPY ./static /webui
COPY .npmrc /webui
COPY package-lock.json /webui
COPY package.json /webui
COPY svelte.config.js /webui
COPY vite.config.js /webui
WORKDIR /webui
RUN npm install -g npm
RUN npm install
CMD ["npm","run","build","&&","ls"]