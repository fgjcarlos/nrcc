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

# Self-heal /data ownership — but only when actually needed, and tolerate
# per-entry EPERMs so one stray immutable file (e.g. a node_modules entry
# pulled in by the host before the first container boot) cannot abort the
# startup. `find ... -not -user node-red` exits 0 even when no match, so
# the path-only `chown` runs only when at least one mismatch exists.
if [ -d /data ]; then
    if find /data -depth -not -user node-red -print -quit 2>/dev/null | grep -q .; then
        # `-depth` so leaves are chowned before their parents, avoiding
        # the parent-first "Operation not permitted" on locked dirs.
        # `-P` = do not follow symlinks (defense vs hostile bind mount;
        # BusyBox and GNU chown both support it — see
        # auditoria/devops-security.md §4.2).
        # Errors from individual entries are intentionally swallowed:
        # a partial chown is strictly better than no chown, and the
        # alternative is a noisy startup that aborts the container.
        chown -R -P node-red:node-red /data 2>/dev/null || true
    fi
fi

# Drop to node-red (uid 1000). BusyBox su clears $HOME/$PATH/$USER by
# default; we want that — the runtime inside nrcc reads DATA_DIR from
# the original env, which su inherits because it does not -l.
# -- separates options from positional args; "$@" forwards any
# container extra args (e.g. --help) to the nrcc binary.
exec su -- node-red -s /bin/sh -c 'exec /usr/local/bin/nrcc "$@"' _ "$@"