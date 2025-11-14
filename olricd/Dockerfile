ARG OLRIC_VERSION=v0.7.1
ARG GOLANG_VERSION=1.25.3

FROM golang:${GOLANG_VERSION} AS builder

ARG OLRIC_VERSION
ARG PLUGIN_REPO=https://github.com/olric-data/olric-kubernetes-plugin.git

WORKDIR /build

RUN git clone --branch ${OLRIC_VERSION} --depth 1 https://github.com/olric-data/olric.git olric

RUN git clone --branch main --depth 1 ${PLUGIN_REPO} plugin

RUN cd plugin && \
    echo >> go.mod && \
    echo "replace github.com/olric-data/olric => ../olric" >> go.mod

WORKDIR /build/plugin
RUN go mod tidy && \
    go build -buildmode=plugin -o /plugin/olric-kubernetes-plugin.so .


FROM ghcr.io/olric-data/olric:${OLRIC_VERSION}

COPY --from=builder /plugin/olric-kubernetes-plugin.so /usr/lib/olric-kubernetes-plugin.so
