FROM node:18.15.0-buster-slim
COPY ./ /webui
WORKDIR /webui
RUN npm install -g npm
RUN npm install
CMD ["npm","run","build"]