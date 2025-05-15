#!/bin/bash

# ----------------------------------------------------------------------------
# Veidemann Cache image entrypoint
# ----------------------------------------------------------------------------

# Create and initialize TLS certificates cache directory
/usr/lib/squid/security_file_certgen -c -s /var/spool/squid/ssl_db -M 4MB
# Set permissions to allow access by proxy user
chown -R proxy:proxy /var/spool/squid/ssl_db

# Start confighandler daemon
confighandler "$@"

tail -F /var/log/squid/access.log 2>/dev/null &
tail -F /var/log/squid/error.log 2>/dev/null &
tail -F /var/log/squid/store.log 2>/dev/null &
tail -F /var/log/squid/cache.log 2>/dev/null &

# Create missing cache directories and exit
/usr/sbin/squid -Nz

/usr/sbin/squid -f /etc/squid/squid.conf -NYC
