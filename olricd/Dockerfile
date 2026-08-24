FROM golang:1.27 AS build

ARG OLRIC_VERSION=v0.7.4
ARG PLUGIN_VERSION=main
ARG PLUGIN_REPO=https://github.com/olric-data/olric-kubernetes-plugin.git

WORKDIR /src

RUN git clone --branch "${OLRIC_VERSION}" --depth 1 \
      https://github.com/olric-data/olric.git olric \
 && git clone --branch "${PLUGIN_VERSION}" --depth 1 \
      "${PLUGIN_REPO}" plugin \
 && go work init ./olric ./plugin \
 && go work sync

RUN CGO_ENABLED=1 go build \
      -C olric \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/app \
      ./cmd/olric-server \
 && CGO_ENABLED=1 go build \
      -C plugin \
      -trimpath \
      -ldflags="-s -w" \
      -buildmode=plugin \
      -o /out/olric-kubernetes-plugin.so \
      . \
 && cp olric/olric-server-docker.yaml /out/olric-server.yaml


FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=build /out/app /app
COPY --from=build /out/olric-server.yaml /etc/olric-server.yaml
COPY --from=build /out/olric-kubernetes-plugin.so /usr/lib/olric-kubernetes-plugin.so

ENTRYPOINT ["/app", "-c", "/etc/olric-server.yaml"]
