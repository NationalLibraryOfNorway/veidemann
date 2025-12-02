FROM golang:1.25 AS build

WORKDIR /build
COPY recorderproxy/ ./recorderproxy/
COPY getlantern-proxy/ ./getlantern-proxy/

WORKDIR /build/recorderproxy
RUN go mod download
# -trimpath remove file system paths from executable
# -ldflags arguments passed to go tool link:
#   -s disable symbol table
#   -w disable DWARF generation
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o proxy .


FROM gcr.io/distroless/base-debian12

COPY --from=build /build/recorderproxy/proxy /

EXPOSE 8080/tcp

ENTRYPOINT ["/proxy"]
