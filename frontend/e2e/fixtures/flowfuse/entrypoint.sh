#!/bin/sh
set -eu

if [ ! -f /data/.nrcc-flowfuse-e2e ]; then
    cp -a /opt/nrcc-flowfuse-e2e/. /data/
    touch /data/.nrcc-flowfuse-e2e
fi

exec /usr/local/bin/nrcc-entrypoint.sh "$@"
