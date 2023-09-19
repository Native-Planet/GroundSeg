FROM node:18.17.0-buster-slim
RUN npm install -g npm@9
COPY ./ /webui
WORKDIR /webui
RUN npm install -g npm
RUN npm install
RUN ls
RUN ls build
CMD ["npm","run","build"]