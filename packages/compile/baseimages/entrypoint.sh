#!/bin/bash
set -eu

# invariant: the architect user joins the group owning the host's docker.sock, so `docker exec -u architect` can drive Docker; re-running on a restart changes nothing
SOCK=/var/run/docker.sock

if [ -S "$SOCK" ]; then
    SOCK_GID=$(stat -c %g "$SOCK")
    GROUP_NAME=$(getent group "$SOCK_GID" | cut -d: -f1 || true)
    if [ -z "$GROUP_NAME" ]; then
        GROUP_NAME=dockerhost
        groupadd -g "$SOCK_GID" "$GROUP_NAME"
    fi
    usermod -aG "$GROUP_NAME" architect
    echo "[architect] docker socket (gid=$SOCK_GID) available to architect via group=$GROUP_NAME"
else
    echo "[architect] warning: /var/run/docker.sock not mounted - architect cannot spawn worlds"
fi

exec "$@"
