package deploy

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"spwn.sh/packages/container/backend"
)

// invariant: the `spwn` line goes first in passwd and group, so a uid the image already gives another user (ubuntu's 1000) still reads back as spwn; files are re-owned on the image's filesystem only, every mount point pruned by its decoded name and matched literally, so no host file changes owner
const followHostUserScript = `set -eu
uid=$1 gid=$2 etc=$3 mountinfo=$4 root=$5
old_uid=$(sed -n 's/^spwn:[^:]*:\([0-9]*\):.*/\1/p' "$etc/passwd")
old_gid=$(sed -n 's/^spwn:[^:]*:[0-9]*:\([0-9]*\):.*/\1/p' "$etc/passwd")
[ -n "$old_uid" ] || exit 3
[ "$old_uid:$old_gid" = "$uid:$gid" ] && exit 0
trap 'exit 4' EXIT
{ sed -n "s/^spwn:\([^:]*\):[0-9]*:[0-9]*:/spwn:\1:$uid:$gid:/p" "$etc/passwd"; grep -v '^spwn:' "$etc/passwd"; } > "$etc/passwd.spwn"
{ sed -n "s/^spwn:\([^:]*\):[0-9]*:/spwn:\1:$gid:/p" "$etc/group"; grep -v '^spwn:' "$etc/group"; } > "$etc/group.spwn"
cat "$etc/passwd.spwn" > "$etc/passwd"
cat "$etc/group.spwn" > "$etc/group"
rm "$etc/passwd.spwn" "$etc/group.spwn"
set --
while read -r _ _ _ _ mnt _; do
	mnt=$(printf '%b' "$(printf '%s' "$mnt" | sed 's/\\\([0-7][0-7][0-7]\)/\\0\1/g')")
	[ "$mnt" = / ] && continue
	set -- "$@" -path "$(printf '%s' "$mnt" | sed 's/[][*?\\]/\\&/g')" -prune -o
done < "$mountinfo"
find "$root" -xdev "$@" \( -uid "$old_uid" -exec chown -h "$uid" {} + -o -true \) -gid "$old_gid" -exec chgrp -h "$gid" {} +
trap - EXIT
`

const followHostUserRemapFailed = 4

// FollowHostUser gives the world's `spwn` user the host user's uid and gid,
// so the agent owns what the host user owns in its bind-mounted workspaces.
func FollowHostUser(ctx context.Context, be backend.Backend, containerID string, uid, gid int) error {
	var output bytes.Buffer
	code, err := be.Exec(ctx, containerID, backend.ExecConfig{
		Cmd:    []string{"sudo", "-n", "sh", "-c", followHostUserScript, "sh", strconv.Itoa(uid), strconv.Itoa(gid), "/etc", "/proc/self/mountinfo", "/"},
		Stdout: &output,
		Stderr: &output,
	})
	if err != nil {
		return fmt.Errorf("follow the host user %d:%d: %w", uid, gid, err)
	}
	switch code {
	case 0:
		return nil
	case followHostUserRemapFailed:
		return fmt.Errorf("follow the host user %d:%d: %s", uid, gid, strings.TrimSpace(output.String()))
	default:
		return fmt.Errorf("the world image needs a spwn user and passwordless sudo on a Linux host (exit %d: %s)", code, strings.TrimSpace(output.String()))
	}
}
