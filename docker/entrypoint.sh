#!/bin/sh
# nrcc container entrypoint.
#
# Self-heal ownership of /data so the runtime, which runs as the
# unprivileged `node-red` user, can write into volumes that Docker
# materialised as root-owned (see issue #484).
#
# The chown must run as root; the nrcc binary itself does not perform
# privilege drop, so we hand off to `su` before exec'ing it. BusyBox
# `setpriv` on this base image ships without --reuid/--regid, and
# `runuser` is not installed.
set -eu

if [ -d /data ]; then
    chown -R node-red:node-red /data
fi

# Drop to node-red (uid 1000). BusyBox su clears $HOME/$PATH/$USER by
# default; we want that — the runtime inside nrcc reads DATA_DIR from
# the original env, which su inherits because it does not -l.
exec su node-red -s /bin/sh -c 'exec /usr/local/bin/nrcc'
