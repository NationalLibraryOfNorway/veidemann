FROM docker.io/olricio/olric-kubernetes-plugin:v0.5.4 AS plugin

FROM ghcr.io/olric-data/olric:v0.7.0

# Copy the cloud plugin from the previous stage
COPY --from=plugin \
/usr/lib/olric-kubernetes-plugin.so \
/usr/lib/olric-kubernetes-plugin.so
