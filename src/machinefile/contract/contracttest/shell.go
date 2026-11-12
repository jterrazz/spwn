package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

func Shell(t *testing.T, lines string, open func(t *testing.T, fails bool) contract.Shell) {
	// value: Shell's suite: lines that end well read exit 0; lines whose
	// shell fails read its non-zero exit and its words on stderr, never an
	// error.
	ctx := context.Background()
	t.Run("lines that end well read exit 0", func(t *testing.T) {
		got, err := open(t, false).Run(ctx, lines)
		if err != nil || got.ExitCode != 0 {
			t.Errorf("Run = %+v, %v; want exit 0", got, err)
		}
	})
	t.Run("lines that fail read their exit and the shell's words", func(t *testing.T) {
		got, err := open(t, true).Run(ctx, lines)
		if err != nil || got.ExitCode == 0 || got.Stderr == "" {
			t.Errorf("Run = %+v, %v; want a non-zero exit with stderr", got, err)
		}
	})
}
