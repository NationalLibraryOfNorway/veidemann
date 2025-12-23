FROM docker.io/golang:1.25-alpine AS build-helpers

WORKDIR /go/src/helpers

COPY helpers/go.mod helpers/go.sum ./
RUN go mod download

COPY helpers ./
RUN CGO_ENABLED=0 go install ./...


FROM docker.io/alpine:3.23 AS generate-certificates

RUN apk add --no-cache gnutls-utils

COPY cert.cfg /
RUN certtool --generate-privkey --outfile tls.key \
    && certtool --generate-self-signed \
        --load-privkey tls.key \
        --template cert.cfg \
        --outfile tls.crt


FROM docker.io/ubuntu:noble-20251013

ENV TZ=UTC
ENV DNS_SERVERS="8.8.8.8 8.8.4.4"

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        tini \
        squid-openssl \
        ca-certificates \
        tzdata; \
    \
    # Squid overrides via conf.d
    mkdir -p /etc/squid/conf.d; \
    printf '%s\n' \
      '# Avoid relying on system RLIMIT_NOFILE. See LP: #1978272' \
      'max_filedescriptors 65536' \
      > /etc/squid/conf.d/20-fdlimit.conf; \
    \
    /usr/sbin/squid --version; \
    rm -rf /var/lib/apt/lists/*

# Persist cache/spool (ssl_db + cache dirs live here)
VOLUME /var/spool/squid

# Helpers
COPY --from=build-helpers /go/bin/confighandler /go/bin/loghelper /go/bin/storeid /usr/local/sbin/

# TLS material (default self-signed; allow users to mount their own)
RUN mkdir -p /ca-certificates
COPY --from=generate-certificates --chown=proxy:proxy /tls.key /tls.crt /ca-certificates/
COPY ffdhe2048.txt /ca-certificates/dhparams.pem

# Squid config + entrypoint
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

COPY squid.conf squid.conf.template squid-balancer.conf.template /etc/squid/

EXPOSE 3128

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/entrypoint.sh"]
