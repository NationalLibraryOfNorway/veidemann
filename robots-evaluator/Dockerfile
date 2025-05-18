FROM golang:latest AS build

ARG VERSION
ARG COMMIT
ARG DATE

WORKDIR /src/
COPY . /src/
RUN go mod download
RUN CGO_ENABLED=0 go build -o /usr/bin/app \
-ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"


FROM gcr.io/distroless/base-debian12
COPY --from=build /usr/bin/app /usr/bin/app

ENTRYPOINT ["/usr/bin/app"]
