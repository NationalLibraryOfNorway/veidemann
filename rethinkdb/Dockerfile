FROM golang:1.27 AS build

WORKDIR /src

COPY probe/go.mod probe/go.sum ./
RUN go mod download

COPY probe/ ./

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/rethinkdb-probe .

WORKDIR /src/startup
COPY startup/ ./
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/rethinkdb-startup .


FROM rethinkdb:2.4.4-bookworm-slim

RUN apt-get update && \
    apt-get -y install --no-install-recommends tini && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /out/rethinkdb-probe /rethinkdb-probe
COPY --from=build /out/rethinkdb-startup /rethinkdb-startup

ENTRYPOINT ["/usr/bin/tini", "--", "/rethinkdb-startup"]
