FROM golang:1.26 AS build

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
# -trimpath remove file system paths from executable
# -ldflags arguments passed to go tool link:
#   -s disable symbol table
#   -w disable DWARF generation
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o proxy .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o generate-mitm-cert ./cmd/generate-mitm-cert


FROM gcr.io/distroless/base-debian12

COPY --from=build /build/proxy /
COPY --from=build /build/generate-mitm-cert /

EXPOSE 8080/tcp

ENTRYPOINT ["/proxy"]
