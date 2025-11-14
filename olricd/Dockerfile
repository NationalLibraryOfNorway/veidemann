FROM golang:1.25 AS build

ARG OLRIC_VERSION=v0.7.1
ARG PLUGIN_REPO=https://github.com/olric-data/olric-kubernetes-plugin.git

WORKDIR /workspace

RUN git clone --branch ${OLRIC_VERSION} --depth 1 \
      https://github.com/olric-data/olric.git olric
RUN git clone --branch main --depth 1 ${PLUGIN_REPO} plugin

RUN go work init ./olric ./plugin
RUN go work sync

WORKDIR /workspace/olric
RUN CGO_ENABLED=1 go build -ldflags="-s -w" \
    -o /out/olric-server ./cmd/olric-server

WORKDIR /workspace/plugin
RUN CGO_ENABLED=1 go build -buildmode=plugin \
    -o /out/olric-kubernetes-plugin.so .

RUN cp /workspace/olric/olric-server-docker.yaml /out/olric-server.yaml


FROM gcr.io/distroless/base-debian12

COPY --from=build /out/olric-server /usr/bin/olric-server
COPY --from=build /out/olric-server.yaml /etc/olric-server.yaml
COPY --from=build /out/olric-kubernetes-plugin.so /usr/lib/olric-kubernetes-plugin.so
EXPOSE 3320 3322
ENTRYPOINT ["/usr/bin/olric-server", "-c", "/etc/olric-server.yaml"]
