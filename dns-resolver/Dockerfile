FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/app .


FROM gcr.io/distroless/static-debian12:nonroot

# dns port
EXPOSE 1053 1053/udp
# veidemann.api.dnsresolver.Resolve
EXPOSE 8053
# prometheus metrics
EXPOSE 9153
# readiness
EXPOSE 8181

ENV DNS_SERVER=8.8.8.8 \
    OLRIC_ADDRESS=localhost:3320 \
    OLRIC_DMAP=dns-resolver-archivingcache \
    CONTENT_WRITER_HOST=veidemann-contentwriter \
    CONTENT_WRITER_PORT=8080 \
    LOG_WRITER_HOST=veidemann-log-service \
    LOG_WRITER_PORT=8080

COPY --from=build /out/app /usr/local/sbin/app
COPY --chown=nonroot:nonroot Corefile.docker /Corefile

ENTRYPOINT ["app"]
CMD ["-conf", "/Corefile"]
