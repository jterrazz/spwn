package machines

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: a folder of the Mac shared live with one of its VMs, at a folder
// of the guest: the Mac's a full path, the guest's a path from `/` or
// `~/` (the VM user's home); read-only when the VM may not write to it.

type Mount struct {
	Computer string `json:"computer"`
	Guest    string `json:"guest"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// value: the folders a start shares, and why they are not its
// Machinefile's when they are not: one line for the owner, empty when they
// are.

type MountsRead struct {
	Mounts []Mount
	Why    string
}

func MountTag(i int) string {
	// value: the tag the hypervisor shares the mount at index i under, and
	// the guest mounts it by.
	return "station-" + strconv.Itoa(i+1)
}

// invariant: the guest's mounts follow the ones the VM booted with: each
// station line of /etc/fstab (its tag first, `nofail`) is the one a mount
// asks, a line no mount asks any more is taken out and its folder
// unmounted, and every mount is mounted at its guest folder, made for the
// VM user when missing. It runs as root and may run again at once: it
// changes nothing that already reads as asked. The engine's mount step
// and a first start's may run at once, so a run holds a lock for all it
// does, and replaces /etc/fstab by a rename, never leaving it half
// written.
const mountScript = `set -e
fstab=/etc/fstab
lock=/run/station-mounts.lock
home=$1
user=$2
shift 2
exec 9>"$lock"
flock 9
nl='
'
want=
while [ $# -ge 3 ]; do
  at=$2
  case $at in
    '~') at=$home ;;
    '~/'*) at=$home/${at#'~/'} ;;
  esac
  want=$want"$1 $at virtiofs $3,nofail 0 0"$nl
  shift 3
done
have=$(grep '^station-[0-9][0-9]* ' "$fstab" || true)
while read -r tag at rest; do
  [ -n "$tag" ] || continue
  case $nl$want in *"$nl$tag $at $rest$nl"*) continue ;; esac
  if mountpoint -q "$at"; then umount "$at" 2>/dev/null || umount -l "$at"; fi
done <<EOF
$have
EOF
if [ "$have" != "${want%"$nl"}" ]; then
  kept=$(grep -v '^station-[0-9][0-9]* ' "$fstab" || true)
  next=$(mktemp "$fstab.station.XXXXXX")
  trap 'rm -f "$next"' EXIT
  printf '%s\n%s' "$kept" "$want" > "$next"
  chmod 644 "$next"
  mv -f "$next" "$fstab"
fi
while read -r tag at rest; do
  [ -n "$tag" ] || continue
  if [ ! -d "$at" ]; then
    mkdir -p "$at"
    chown "$user:" "$at"
  fi
  mountpoint -q "$at" || mount "$at" || mountpoint -q "$at"
done <<EOF
$want
EOF
`

// value: the line and the script (its stdin) that bring a guest's mounts
// to these: each mount's tag, guest folder and mode are the line's
// arguments, quoted, never the script's text.

func MountLine(mounts []Mount) (line, script string) {
	args := make([]string, 0, 1+3*len(mounts))
	args = append(args, `bash -s -- "$HOME" "$(id -un)"`)
	for i, m := range mounts {
		mode := "rw"
		if m.ReadOnly {
			mode = "ro"
		}
		args = append(args, quoted(MountTag(i)), quoted(m.Guest), mode)
	}
	return catalog.AsRoot(strings.Join(args, " ")), mountScript
}

func quoted(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }
