#!/bin/bash
set -euo pipefail

TAIL_PID=""
CONF_PID=""
SQUID_PID=""

# Ensure runtime dirs exist and are owned correctly
mkdir -p /run/squid /var/spool/squid
chown -R proxy:proxy /run/squid /var/spool/squid

# Ensure cache_log file exists and is writable
touch /run/squid/cache.log
chown proxy:proxy /run/squid/cache.log
chmod 0644 /run/squid/cache.log

# Forward cache_log to container stderr
tail -n 0 -F /run/squid/cache.log >&2 &
TAIL_PID=$!

# shellcheck disable=SC2329
shutdown() {
  echo "caught signal, shutting down..." >&2

  # Ask squid to stop gracefully first
  if [ -n "${SQUID_PID}" ] && kill -0 "${SQUID_PID}" 2>/dev/null; then
    squid -k shutdown 2>/dev/null || kill -TERM "${SQUID_PID}" 2>/dev/null || true
  fi

  # Stop confighandler
  if [ -n "${CONF_PID}" ] && kill -0 "${CONF_PID}" 2>/dev/null; then
    kill -TERM "${CONF_PID}" 2>/dev/null || true
  fi

  # Stop tailing cache_log
  if [ -n "${TAIL_PID}" ] && kill -0 "${TAIL_PID}" 2>/dev/null; then
    kill -TERM "${TAIL_PID}" 2>/dev/null || true
  fi

  # Wait for processes (if they exist)
  [ -n "${SQUID_PID}" ] && wait "${SQUID_PID}" 2>/dev/null || true
  [ -n "${CONF_PID}" ] && wait "${CONF_PID}" 2>/dev/null || true
  [ -n "${TAIL_PID}" ] && wait "${TAIL_PID}" 2>/dev/null || true
}

trap shutdown INT TERM EXIT

# --- init ssl_db ---
if [ ! -d /var/spool/squid/ssl_db ] || [ -z "$(ls -A /var/spool/squid/ssl_db 2>/dev/null)" ]; then
  /usr/lib/squid/security_file_certgen -c -s /var/spool/squid/ssl_db -M 4MB
fi
chown -R proxy:proxy /var/spool/squid/ssl_db

# Start confighandler (foreground in background job)
confighandler "$@" --ready-file /run/squid/confighandler.ready &
CONF_PID=$!

# Wait for initial config render
for _ in {1..60}; do
  if [ -f /run/squid/confighandler.ready ]; then
    break
  fi
  sleep 0.2
done

if [ ! -f /run/squid/confighandler.ready ]; then
  echo "confighandler did not become ready" >&2
  exit 1
fi

# Ensure cache dirs exist (now that config is rendered, if it affects cache_dir)
 /usr/sbin/squid -Nz

# Start squid (foreground)
squid -f /etc/squid/squid.conf -N &
SQUID_PID=$!

# If either squid or confighandler exits unexpectedly, shut down the other.
wait -n "$SQUID_PID" "$CONF_PID"
exit 0
