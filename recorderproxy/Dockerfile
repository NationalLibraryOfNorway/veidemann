FROM golang:1.25 AS build

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go install -v ./...


FROM gcr.io/distroless:base-debian12

COPY --from=build /go/bin/recorderproxy /

EXPOSE 8080
ENV CACHE_HOST=localhost \
    CACHE_PORT=9999 \
    PORT=9900 \
    PROXY_COUNT=10 \
    DNS_RESOLVER_HOST=localhost \
    DNS_RESOLVER_PORT=7777 \
    CONTENT_WRITER_HOST=localhost \
    CONTENT_WRITER_PORT=7778 \
    BROWSER_CONTROLLER_HOST=localhost \
    BROWSER_CONTROLLER_PORT=7779 \
    CA="" \
    CA_KEY=""

ENTRYPOINT ["/recorderproxy"]
