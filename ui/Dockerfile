FROM node:18.15.0-buster-slim

COPY ./ /webui

WORKDIR /webui
RUN npm install -g npm
RUN npm install

RUN npm run build
RUN rm -r src
RUN rm -r static
RUN rm svelte.config.js
RUN rm vite.config.js
RUN rm -r node_modules

WORKDIR /webui/build
ENV NODE_ENV=production

EXPOSE 3000
ENTRYPOINT ["node", "index.js"]
