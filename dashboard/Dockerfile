FROM node:25-bookworm-slim AS build

ARG VERSION

WORKDIR /usr/src/app

COPY . .

RUN npm ci --quiet --legacy-peer-deps
RUN sed -i "s/version: ''/version: '${VERSION}'/" src/environments/*.ts
RUN npm run build


FROM nginx:stable-alpine-slim
LABEL maintainer="nettarkivet@nb.no"

COPY --from=build /usr/src/app/dist /usr/share/nginx
COPY --from=build /usr/src/app/nginx/default.conf /etc/nginx/conf.d/
