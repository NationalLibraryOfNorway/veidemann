FROM golang:1.27 AS build

ARG VERSION
ARG COMMIT
ARG DATE

WORKDIR /src

COPY go.mod go.sum ./
COPY third_party/chromedp/go.mod third_party/chromedp/go.sum ./third_party/chromedp/
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /out/app .


FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=build /out/app /app

ENTRYPOINT ["/app"]
