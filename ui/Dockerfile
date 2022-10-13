FROM node:18-buster-slim

COPY ./ /webui

WORKDIR /webui
RUN npm install

RUN npm run build
RUN rm -r src
RUN rm -r static
RUN rm svelte.config.js
RUN rm vite.config.js

WORKDIR /webui/build
EXPOSE 3000
ENTRYPOINT ["node", "index.js"]
