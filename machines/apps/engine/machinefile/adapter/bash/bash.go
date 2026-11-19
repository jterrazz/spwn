package bash

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: the owner's lines are the driver's argument, never pasted
// into it. The driver evals one complete command at a time (a line, or
// the lines a multi-line construct, a heredoc or a trailing `\` holds, as
// `bash -n` reads them), all in one shell under -e and pipefail, so a cd
// and an export carry to the next line; a command whose status is not
// zero ends the run with that status, even where -e lets it pass (the
// left of `&&`, a pipe), and its first line is said on stderr. A line's
// trailing CR is dropped, a whole-line comment never continues on a
// trailing `\`, and the owner's commands do not inherit the descriptor
// the lines are read from. It calls bash's builtins alone, so it runs on
// any PATH, bash 3.2 included.
const driver = `set -o pipefail
__station_nl='
'
__station_part=
__station_said() {
  __station_s=$?
  [ "$__station_s" -eq 0 ] || [ -z "$__station_part" ] || printf 'station: exit %s at: %s\n' "$__station_s" "${__station_part%%"$__station_nl"*}" >&2
}
trap __station_said EXIT
__station_cr=$(printf '\r')
while IFS= read -r __station_line <&3 || [ -n "$__station_line" ]; do
  __station_line=${__station_line%"$__station_cr"}
  __station_part=$__station_part$__station_line$__station_nl
  case $__station_line in
  *\\) case ${__station_line#"${__station_line%%[![:space:]]*}"} in '#'*) ;; *) continue ;; esac ;;
  esac
  [ -z "$("$BASH" -n -c "$__station_part" 2>&1)" ] || continue
  eval "$__station_part" 3<&-
  __station_s=$?
  [ "$__station_s" -eq 0 ] || exit "$__station_s"
  __station_part=
done 3<<<"$1"
[ -z "$__station_part" ] || eval "$__station_part" 3<&-
`

type Bash struct {
	run  local.Runner
	home string
}

var _ contract.Shell = (*Bash)(nil)

func New(run local.Runner, home string) *Bash { return &Bash{run: run, home: home} }

func (b *Bash) Run(ctx context.Context, lines string) (contract.ShellRun, error) {
	c := local.Command("bash", "-e", "-c", driver, "station", lines)
	c.Dir, c.Timeout = b.home, catalog.RunBound
	out, err := b.run.Output(ctx, c)
	if err != nil {
		return contract.ShellRun{}, err
	}
	return contract.ShellRun{ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr}, nil
}
