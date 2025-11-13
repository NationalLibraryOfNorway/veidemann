FROM golang:1.25-bookworm AS probe

WORKDIR /go/src/probe

COPY probe/* ./
RUN go mod download
RUN CGO_ENABLED=0 go install -trimpath -ldflags "-s -w"


FROM rethinkdb:2.4.4-bookworm-slim

LABEL maintainer="Marius André Elsfjordstrand Beck <marius.beck@nb.no>"

ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-amd64 /usr/local/bin/tini
RUN chmod +x /usr/local/bin/tini

COPY --from=probe /go/bin/probe /rethinkdb-probe
COPY ./run.sh /
RUN chmod u+x /run.sh

EXPOSE 28015 29015 8080

ENTRYPOINT ["tini", "/run.sh"]
