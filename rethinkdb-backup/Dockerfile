FROM docker.io/library/rethinkdb:2.4.4-bookworm-slim

LABEL maintainer="marius.beck@nb.no"

ENV VIRTUAL_ENV=/opt/venv
ENV PATH="${VIRTUAL_ENV}/bin:${PATH}"

RUN apt-get update && \
    apt-get -y install --no-install-recommends python3 python3-venv && \
    python3 -m venv "${VIRTUAL_ENV}" && \
    pip install --no-cache-dir rethinkdb && \
    apt-get purge -y --auto-remove python3-venv && \
    rm -rf /var/lib/apt/lists/*

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

USER rethinkdb

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["--help"]
