FROM golang:1.25-bookworm AS probe

WORKDIR /go/src/probe

COPY probe/* ./
RUN go mod download
RUN CGO_ENABLED=0 go install -trimpath -ldflags "-s -w"


FROM rethinkdb:2.4.4-bookworm-slim

LABEL maintainer="Marius André Elsfjordstrand Beck <marius.beck@nb.no>"

RUN apt-get update && \
    apt-get -y install --no-install-recommends coreutils libc-bin tini && \
    rm -rf /var/lib/apt/lists/*

COPY --from=probe /go/bin/probe /rethinkdb-probe
COPY ./run.sh /
RUN chmod u+x /run.sh

EXPOSE 28015 29015 8080

ENTRYPOINT ["/usr/bin/tini", "--", "/run.sh"]
