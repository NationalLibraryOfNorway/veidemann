FROM golang:1.24 AS build

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# -trimpath remove file system paths from executable
# -ldflags arguments passed to go tool link:
#   -s disable symbol table
#   -w disable DWARF generation
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o app

FROM gcr.io/distroless/base-debian12

COPY --from=build /build/app /app
COPY --from=build /build/lua /lua

ENTRYPOINT ["/app"]
