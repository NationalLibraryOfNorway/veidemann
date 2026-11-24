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


FROM nginx/nginx-unprivileged:1.30.8-alpine-slim@sha256:04c98c78c1de60ddeada00d7d2207dcd146ecd07b61b9941c9a5799cfb258d28
LABEL maintainer="nettarkivet@nb.no"

# The pinned base ships zlib 1.3.2-r1, affected by CVE-2026-4738.
USER root
RUN apk add --no-cache 'zlib>=1.4-r0'
USER 101:101

COPY --from=build /usr/src/app/dist/ /usr/share/nginx/
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
