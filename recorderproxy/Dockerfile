FROM golang:1.27 AS build

ARG VERSION
ARG COMMIT
ARG DATE

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/app \
    .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/generate-mitm-cert \
    ./cmd/generate-mitm-cert


FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=build /out/app /app
COPY --from=build /out/generate-mitm-cert /generate-mitm-cert

ENTRYPOINT ["/app"]
