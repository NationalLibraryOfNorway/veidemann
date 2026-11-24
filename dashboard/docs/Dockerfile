FROM alpine:latest as build

# install git
RUN apk add --no-cache git

ARG HUGO_VERSION=0.98.0
ARG HUGO_BINARY=hugo_${HUGO_VERSION}_Linux-64bit.tar.gz
ARG SITE_DIR='/usr/share/site'

# fetch hugo
RUN wget -O /tmp/hugo.tar.gz \
    https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_${HUGO_VERSION}_Linux-64bit.tar.gz \
    && tar xvzf /tmp/hugo.tar.gz \
    && install hugo /usr/local/bin/hugo \
    && mkdir ${SITE_DIR}

WORKDIR ${SITE_DIR}

COPY . .

# install theme
RUN cd themes && git clone --depth 1 https://github.com/matcornic/hugo-theme-learn.git
# generate site
RUN hugo -d out

FROM nginx/nginx-unprivileged:1.30.8-alpine@sha256:637c35e46af7bc0dd3d3bcf7162e004e13ab3b370fde7cd416e2ac410b5a985a
LABEL maintainer="nettarkivet@nb.no"

# The pinned base ships zlib 1.3.2-r1, affected by CVE-2026-4738.
USER root
RUN apk add --no-cache 'zlib>=1.4-r0'
USER 101:101

COPY --from=build /usr/share/site/out /usr/share/nginx/html/
COPY --from=build /usr/share/site/nginx/default.conf /etc/nginx/conf.d/
