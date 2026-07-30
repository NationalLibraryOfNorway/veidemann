#!/usr/bin/env bash
set -euo pipefail

POD_NAMESPACE=${POD_NAMESPACE:-default}
POD_NAME=${POD_NAME:?POD_NAME must be set}
POD_IP=${POD_IP:-127.0.0.1}

RETHINKDB_PASSWORD=${RETHINKDB_PASSWORD:-auto}
RETHINKDB_SERVICE_NAME=${RETHINKDB_SERVICE_NAME:-rethinkdb}
RETHINKDB_CLUSTER_PORT=${RETHINKDB_CLUSTER_PORT:-29015}
RETHINKDB_SEEDS=${RETHINKDB_SEEDS:-}
RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME:-}
RETHINKDB_CLUSTER_DOMAIN=${RETHINKDB_CLUSTER_DOMAIN:-cluster.local}
RETHINKDB_DISCOVERY_ATTEMPTS=${RETHINKDB_DISCOVERY_ATTEMPTS:-5}
RETHINKDB_DISCOVERY_DELAY_SECONDS=${RETHINKDB_DISCOVERY_DELAY_SECONDS:-2}
RETHINKDB_DISCOVERY_CONNECT_TIMEOUT_SECONDS=${RETHINKDB_DISCOVERY_CONNECT_TIMEOUT_SECONDS:-1}

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

# Global default for the StatefulSet name if it is still empty.
RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME:-rethinkdb}

JOIN_ARGS=()

add_join() {
  local host="$1"

  # Add the cluster port while preserving explicit ports and raw IPv6 addresses.
  case "$host" in
  \[*\]:*) ;;
  \[*\]) host="${host}:${RETHINKDB_CLUSTER_PORT}" ;;
  *:*:*) host="[${host}]:${RETHINKDB_CLUSTER_PORT}" ;;
  *:*) ;;
  *) host="${host}:${RETHINKDB_CLUSTER_PORT}" ;;
  esac

  JOIN_ARGS+=(--join "$host")
}

peer_is_reachable() {
  local host="$1"

  # $1 and $2 are expanded by the child Bash process.
  # shellcheck disable=SC2016
  timeout "${RETHINKDB_DISCOVERY_CONNECT_TIMEOUT_SECONDS}" \
    bash -c 'exec 3<>"/dev/tcp/${1}/${2}"' \
    bash "$host" "$RETHINKDB_CLUSTER_PORT" 2>/dev/null
}

discover_statefulset_peers() {
  local service_host="${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.${RETHINKDB_CLUSTER_DOMAIN}"
  local attempt ip _
  local -A attempted=()

  echo "Discovering reachable peers through ${service_host}"

  for ((attempt = 1; attempt <= RETHINKDB_DISCOVERY_ATTEMPTS; attempt++)); do
    attempted=()
    while read -r ip _; do
      [[ -z "$ip" || "$ip" == "$POD_IP" || -n "${attempted[$ip]:-}" ]] && continue
      attempted["$ip"]=1

      if peer_is_reachable "$ip"; then
        echo "Discovered reachable peer ${ip}:${RETHINKDB_CLUSTER_PORT}"
        add_join "$ip"
      fi
    done < <(getent ahosts "$service_host" 2>/dev/null || true)

    if ((${#JOIN_ARGS[@]} > 0)); then
      return
    fi

    if ((attempt < RETHINKDB_DISCOVERY_ATTEMPTS)); then
      echo "No reachable peers found (attempt ${attempt}/${RETHINKDB_DISCOVERY_ATTEMPTS}); retrying"
      sleep "$RETHINKDB_DISCOVERY_DELAY_SECONDS"
    fi
  done

  echo "No reachable peers found; starting as the first available StatefulSet member"
}

CANONICAL_ADDRESS=$POD_IP
if [[ -n "$ORDINAL" ]]; then
  CANONICAL_ADDRESS="${POD_NAME}.${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.${RETHINKDB_CLUSTER_DOMAIN}:${RETHINKDB_CLUSTER_PORT}"
fi

echo "POD_NAME=${POD_NAME}"
echo "POD_NAMESPACE=${POD_NAMESPACE}"
echo "POD_IP=${POD_IP}"
echo "RETHINKDB_SERVICE_NAME=${RETHINKDB_SERVICE_NAME}"
echo "RETHINKDB_STATEFULSET_NAME=${RETHINKDB_STATEFULSET_NAME}"
echo "ORDINAL=${ORDINAL:-<none>}"
echo "RETHINKDB_SEEDS=${RETHINKDB_SEEDS:-<none>}"
echo "CANONICAL_ADDRESS=${CANONICAL_ADDRESS}"
echo "PROXY=${PROXY:-<unset>}"

##
## Build JOIN_ARGS
##

# 1) If explicit seeds are provided, always use those.
if [[ -n "$RETHINKDB_SEEDS" ]]; then
  echo "Using explicit seeds from RETHINKDB_SEEDS"
  IFS=', ' read -r -a seeds <<<"$RETHINKDB_SEEDS"
  for s in "${seeds[@]}"; do
    [[ -z "$s" ]] && continue
    add_join "$s"
  done

# 2) StatefulSet members discover current peers from the headless Service.
# This gives the first pod no seeds during a clean or full bootstrap, while a
# restarting ordinal 0 discovers the other members during a rolling update.
elif [[ -n "$ORDINAL" ]]; then
  discover_statefulset_peers

# 3) Otherwise (non-StatefulSet or ordinal-less pod), default to seed 0.
else
  echo "Non-StatefulSet or ordinal-less pod, using ${RETHINKDB_STATEFULSET_NAME}-0 as default seed"
  host="${RETHINKDB_STATEFULSET_NAME}-0.${RETHINKDB_SERVICE_NAME}.${POD_NAMESPACE}.svc.${RETHINKDB_CLUSTER_DOMAIN}"
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
    --canonical-address "${CANONICAL_ADDRESS}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
else
  echo "Starting RethinkDB server"
  exec rethinkdb \
    --server-name "${SERVER_NAME}" \
    --canonical-address "${CANONICAL_ADDRESS}" \
    --initial-password "${RETHINKDB_PASSWORD}" \
    "${JOIN_ARGS[@]}" \
    "$@"
fi
