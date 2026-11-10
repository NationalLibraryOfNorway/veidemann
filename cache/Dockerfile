FROM golang:1.27 AS build

WORKDIR /src

COPY helpers/go.mod helpers/go.sum ./
RUN go mod download

COPY helpers ./
RUN CGO_ENABLED=0 go install \
    -trimpath \
    -ldflags="-s -w" \
    ./...

FROM debian:sid-slim

ARG SQUID_MAJOR_VERSION=7

ENV TZ=UTC \
    DNS_SERVERS="8.8.8.8 8.8.4.4"

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        squid-openssl \
        tini \
        tzdata; \
    case "${SQUID_MAJOR_VERSION}" in \
      ''|*[!0-9]*) echo "SQUID_MAJOR_VERSION must be a number" >&2; exit 1 ;; \
    esac; \
    squid_version="$(dpkg-query -W -f='${Version}' squid-openssl)"; \
    case "${squid_version#*:}" in \
      "${SQUID_MAJOR_VERSION}".*) ;; \
      *) echo "Expected squid-openssl ${SQUID_MAJOR_VERSION}.x, got ${squid_version}" >&2; exit 1 ;; \
    esac; \
    # Squid overrides via conf.d
    mkdir -p /etc/squid/conf.d; \
    # The Debian fragment contains APT-specific refresh patterns after our
    # catch-all pattern. They are unreachable and use ages Squid must clamp.
    rm -f /etc/squid/conf.d/debian.conf; \
    printf '%s\n' \
      '# Bound Squid descriptor tables independently of container runtime limits.' \
      '# Avoid both a low inherited limit and huge allocations; see LP: #1978272.' \
      'max_filedescriptors 65536' \
      > /etc/squid/conf.d/20-fdlimit.conf; \
    /usr/sbin/squid --version; \
    rm -rf /var/lib/apt/lists/*

COPY --from=build \
    /go/bin/confighandler \
    /go/bin/storeid \
    /usr/local/sbin/

COPY entrypoint.sh /usr/local/bin/
COPY squid.conf squid.conf.template squid-balancer.conf.template /etc/squid/

EXPOSE 3128

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/entrypoint.sh"]
