#!/usr/bin/env bash
set -euo pipefail

POD_NAMESPACE=${POD_NAMESPACE:-default}
POD_NAME=${POD_NAME:?POD_NAME must be set}
POD_IP=${POD_IP:-127.0.0.1}

RETHINKDB_PASSWORD=${RETHINKDB_PASSWORD:-auto}
RETHINKDB_SERVICE_NAME=${RETHINKDB_SERVICE_NAME:-rethinkdb}
# Default STS name = pod name without trailing "-<number>"
RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME:-${POD_NAME%-[0-9]*}}

SERVER_NAME=${POD_NAME//-/_}
ORDINAL="${POD_NAME##*-}"

JOIN_ARGS=()

if [[ "$ORDINAL" != "0" ]]; then
  # Join all earlier ordinals as seeds: 0..(ordinal-1)
  for ((i=0; i<ORDINAL; i++)); do
    host="${RETHINKDB_STATEFULSET_NAME}-${i}.${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.cluster.local"
    JOIN_ARGS+=(--join "${host}:29015")
  done
fi

if [[ -n "${PROXY:-}" ]]; then
  exec rethinkdb \
    proxy \
    --canonical-address "${POD_IP}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
else
  exec rethinkdb \
    --server-name "${SERVER_NAME}" \
    --canonical-address "${POD_IP}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
fi
