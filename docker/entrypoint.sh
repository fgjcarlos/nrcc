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
# Defensive defaults per .agents/skills/bash-defensive-patterns.
# -E: inherit ERR trap into shell functions; -e: exit on error;
# -u: error on unset variables; -o pipefail: pipe fails if any
# stage fails. The script has no pipelines today, but the flags
# are a free safety net for future edits.
set -Eeuo pipefail
# Owner-only default file creation. Files created by the entrypoint
# or by the nrcc binary (e.g. log files) inherit this umask.
umask 0077

if [ -d /data ]; then
    # -P = do not follow symlinks (defense vs hostile bind mount).
    # BusyBox and GNU chown both support -P; see
    # auditoria/devops-security.md §4.2.
    chown -R -P node-red:node-red /data
fi

# Drop to node-red (uid 1000). BusyBox su clears $HOME/$PATH/$USER by
# default; we want that — the runtime inside nrcc reads DATA_DIR from
# the original env, which su inherits because it does not -l.
# -- separates options from positional args; "$@" forwards any
# container extra args (e.g. --help) to the nrcc binary.
exec su -- node-red -s /bin/sh -c 'exec /usr/local/bin/nrcc "$@"' _ "$@"
