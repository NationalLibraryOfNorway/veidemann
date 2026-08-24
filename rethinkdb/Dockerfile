FROM golang:1.27 AS build

WORKDIR /src

COPY probe/go.mod probe/go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/rethinkdb-probe .


FROM rethinkdb:2.4.4-bookworm-slim

RUN apt-get update && \
    apt-get -y install --no-install-recommends coreutils libc-bin tini && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /out/rethinkdb-probe /rethinkdb-probe
COPY ./run.sh /run.sh
RUN chmod u+x /run.sh

ENTRYPOINT ["/usr/bin/tini", "--", "/run.sh"]
