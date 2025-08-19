FROM docker.io/golang:1.24-alpine AS build-helpers

WORKDIR /go/src/helpers

COPY helpers/go.mod helpers/go.sum ./
RUN go mod download

COPY helpers ./
RUN CGO_ENABLED=0 go install ./...


FROM docker.io/alpine:3.22 AS generate-certificates

RUN apk add --no-cache gnutls-utils

COPY cert.cfg /
RUN certtool --generate-privkey --outfile tls.key \
    && certtool --generate-self-signed --load-privkey tls.key --template cert.cfg --outfile tls.crt


FROM docker.io/debian:bookworm-20250811-slim

ENV TZ=UTC
ENV SERVICE_NAME="veidemann-cache"
ENV DNS_SERVERS="8.8.8.8 8.8.4.4"

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get full-upgrade -y; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends squid-openssl ca-certificates tzdata; \
    DEBIAN_FRONTEND=noninteractive apt-get remove --purge --auto-remove -y; \
    rm -rf /var/lib/apt/lists/*; \
    sed -i 's/^#http_access allow localnet$/http_access allow localnet/' /etc/squid/conf.d/debian.conf; \
    echo "# Set max_filedescriptors to avoid using system's RLIMIT_NOFILE. See LP: #1978272" > /etc/squid/conf.d/rock.conf; \
    echo 'max_filedescriptors 65536' >> /etc/squid/conf.d/rock.conf; \
    /usr/sbin/squid --version;

# Use this mount to bring your own certificates
VOLUME /ca-certificates
VOLUME /var/spool/squid

COPY --from=build-helpers /go/bin/confighandler /go/bin/storeid /go/bin/loghelper /usr/local/sbin/
COPY --from=generate-certificates --chown=proxy:proxy /tls.key /tls.crt /ca-certificates/
ADD https://ssl-config.mozilla.org/ffdhe2048.txt /ca-certificates/dhparams.pem
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY squid.conf squid.conf.template squid-balancer.conf.template /etc/squid/

EXPOSE 3128

ENTRYPOINT [ "entrypoint.sh"]
