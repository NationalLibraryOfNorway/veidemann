#!/usr/bin/env bash
set -euo pipefail

POD_NAMESPACE=${POD_NAMESPACE:-default}
POD_NAME=${POD_NAME:?POD_NAME must be set}
POD_IP=${POD_IP:-127.0.0.1}

RETHINKDB_PASSWORD=${RETHINKDB_PASSWORD:-auto}
RETHINKDB_SERVICE_NAME=${RETHINKDB_SERVICE_NAME:-rethinkdb}
RETHINKDB_CLUSTER_PORT=${RETHINKDB_CLUSTER_PORT:-29015}
RETHINKDB_SEEDS=${RETHINKDB_SEEDS:-}
RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME:-}   # we'll default later

SERVER_NAME=${POD_NAME//-/_}

# Detect ordinal only for names like foo-0, foo-1, ...
ORDINAL=""
if [[ "$POD_NAME" =~ ^(.+)-([0-9]+)$ ]]; then
  # If not explicitly set, base name = prefix
  if [[ -z "$RETHINKDB_STATEFULSET_NAME" ]]; then
    RETHINKDB_STATEFULSET_NAME="${BASH_REMATCH[1]}"
  fi
  ORDINAL="${BASH_REMATCH[2]}"
fi

# Global default for statefulset name if still empty (your "rethinkdb" default)
RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME:-rethinkdb}

JOIN_ARGS=()

add_join() {
  local host="$1"
  # Add default port if none given
  if [[ "$host" != *:* ]]; then
    host="${host}:${RETHINKDB_CLUSTER_PORT}"
  fi
  JOIN_ARGS+=(--join "$host")
}

echo "POD_NAME=${POD_NAME}"
echo "POD_NAMESPACE=${POD_NAMESPACE}"
echo "POD_IP=${POD_IP}"
echo "RETHINKDB_SERVICE_NAME=${RETHINKDB_SERVICE_NAME}"
echo "RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME}"
echo "ORDINAL=${ORDINAL:-<none>}"
echo "RETHINKDB_SEEDS=${RETHINKDB_SEEDS:-<none>}"
echo "PROXY=${PROXY:-<unset>}"

##
## Build JOIN_ARGS
##

# 1) If explicit seeds are provided, always use those.
if [[ -n "$RETHINKDB_SEEDS" ]]; then
  echo "Using explicit seeds from RETHINKDB_SEEDS"
  IFS=', ' read -r -a seeds <<< "$RETHINKDB_SEEDS"
  for s in "${seeds[@]}"; do
    [[ -z "$s" ]] && continue
    add_join "$s"
  done

# 2) Otherwise, if this is a StatefulSet pod with an ordinal > 0, join all earlier ordinals.
elif [[ -n "$ORDINAL" && "$ORDINAL" != "0" ]]; then
  echo "StatefulSet pod with ordinal ${ORDINAL}, joining all earlier ordinals"
  for ((i=0; i<ORDINAL; i++)); do
    host="${RETHINKDB_STATEFULSET_NAME}-${i}.${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.cluster.local"
    add_join "$host"
  done

# 3) Otherwise (non-StatefulSet or ordinal-less pod), default to seed 0.
else
  echo "Non-StatefulSet or ordinal-less pod, using ${RETHINKDB_STATEFULSET_NAME}-0 as default seed"
  host="${RETHINKDB_STATEFULSET_NAME}-0.${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.cluster.local"
  add_join "$host"
fi

# In proxy mode we MUST have at least one join endpoint.
if [[ -n "${PROXY:-}" && ${#JOIN_ARGS[@]} -eq 0 ]]; then
  echo "ERROR: PROXY is set but no join endpoints were computed."
  echo "Set RETHINKDB_SEEDS or check RETHINKDB_SERVICE_NAME/RETHINKDB_STATEFULSET_NAME."
  exit 1
fi

echo "Final join args: ${JOIN_ARGS[*]}"

if [[ -n "${PROXY:-}" ]]; then
  echo "Starting RethinkDB in proxy mode"
  exec rethinkdb \
    proxy \
    --canonical-address "${POD_IP}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
else
  echo "Starting RethinkDB server"
  exec rethinkdb \
    --server-name "${SERVER_NAME}" \
    --canonical-address "${POD_IP}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
fi
