# Copyright 2026 National Library of Norway.
# SPDX-License-Identifier: Apache-2.0

FROM nginx/nginx-unprivileged:1.30.8-alpine@sha256:637c35e46af7bc0dd3d3bcf7162e004e13ab3b370fde7cd416e2ac410b5a985a

# The pinned base ships zlib 1.3.2-r1, affected by CVE-2026-4738.
USER root
RUN apk add --no-cache 'zlib>=1.4-r0' \
    && install -d -m 0700 -o 101 -g 101 /etc/nginx/certs
USER 101:101

COPY default.conf /etc/nginx/conf.d/default.conf
COPY site/ /usr/share/nginx/html/
