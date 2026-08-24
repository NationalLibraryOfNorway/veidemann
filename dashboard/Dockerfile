FROM node:26-trixie-slim AS build

ARG VERSION

WORKDIR /usr/src/app

COPY package.json package-lock.json ./
RUN npm ci --quiet --no-fund --no-audit

COPY angular.json tsconfig.json tsconfig.app.json ./
COPY src/ src/
COPY public/ public/

RUN sed -i "s/version: ''/version: '${VERSION}'/" src/environments/*.ts \
    && npm run build


FROM nginx:stable-alpine-slim
LABEL maintainer="nettarkivet@nb.no"

COPY --from=build /usr/src/app/dist/ /usr/share/nginx/
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
