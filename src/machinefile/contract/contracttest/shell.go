package contracttest

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

func Shell(t *testing.T, open func(t *testing.T, fails bool) contract.Shell) {
	// value: Shell's suite: lines that end well read exit 0; lines whose
	// first line fails read its non-zero exit and the shell's words on
	// stderr, the last naming that line, never an error.
	ctx := context.Background()
	t.Run("lines that end well read exit 0", func(t *testing.T) {
		got, err := open(t, false).Run(ctx, "cd ~\ntrue\n")
		if err != nil || got.ExitCode != 0 {
			t.Errorf("Run = %+v, %v; want exit 0", got, err)
		}
	})
	t.Run("lines that fail read their exit and the line that failed", func(t *testing.T) {
		got, err := open(t, true).Run(ctx, "false && true\necho never\n")
		if err != nil || got.ExitCode != 1 || !strings.HasSuffix(got.Stderr, "station: exit 1 at: false && true\n") {
			t.Errorf("Run = %+v, %v; want exit 1, stderr ending with the line that failed", got, err)
		}
	})
}
