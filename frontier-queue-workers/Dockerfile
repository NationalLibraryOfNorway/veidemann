FROM golang:1.25 AS build

WORKDIR /build

COPY . .
RUN go mod download

# -trimpath remove file system paths from executable
# -ldflags arguments passed to go tool link:
#   -s disable symbol table
#   -w disable DWARF generation
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w"

FROM gcr.io/distroless/base-debian12

COPY --from=build /build/frontier-queue-workers /app
COPY --from=build /build/lua /lua

ENTRYPOINT ["/app"]
