#!/bin/bash
# Architect container entrypoint.
#
# The container runs as root (needed for bind-mount file access), but
# Claude Code runs as the unprivileged `architect` user because it
# refuses --dangerously-skip-permissions when invoked as root.
#
# The `architect` user needs access to /var/run/docker.sock so it can
# spawn worlds on the host's Docker daemon. The socket is mounted from
# the host with its native owner (root:docker) and GID — which varies
# per host (Docker Desktop / OrbStack / Linux). We can't hardcode the
# GID into the image, so at startup we read the socket's actual GID,
# create or reuse a matching group inside the container, and add the
# architect user to it. After this, `docker exec -u architect` sessions
# have read+write access to the socket.
#
# Idempotent: safe to re-run if the container restarts. No-ops cleanly
# when the socket isn't mounted (non-architect spawns won't have it).
set -eu

SOCK=/var/run/docker.sock

if [ -S "$SOCK" ]; then
    SOCK_GID=$(stat -c %g "$SOCK")
    # If a group with that GID already exists in the container, use it.
    # Otherwise, create a new group pinned to that GID.
    GROUP_NAME=$(getent group "$SOCK_GID" | cut -d: -f1 || true)
    if [ -z "$GROUP_NAME" ]; then
        GROUP_NAME=dockerhost
        groupadd -g "$SOCK_GID" "$GROUP_NAME"
    fi
    # Add architect to that group (idempotent — usermod -aG is safe to re-run).
    usermod -aG "$GROUP_NAME" architect
    echo "[architect] docker socket (gid=$SOCK_GID) available to architect via group=$GROUP_NAME"
else
    echo "[architect] warning: /var/run/docker.sock not mounted — architect cannot spawn worlds"
fi

exec "$@"
