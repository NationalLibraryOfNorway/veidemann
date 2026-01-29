FROM docker.io/golang:1.25.6 AS builder

WORKDIR /build

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build  -trimpath -ldflags "-s -w"


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/ooshandler /

EXPOSE 9301 50052
VOLUME "/data"

ENTRYPOINT ["/ooshandler"]
