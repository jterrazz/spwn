package kit

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/output"
)

func TestASpinnersStepAtATerminalWithholdsItsToken(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	requiredBy := "required-by "
	for name, detail := range map[string]string{
		"its spinner line": requiredBy + token,
		"its ahead line":   call.Asks + "\n" + requiredBy + token,
	} {
		var tripped []string
		secret.Arm(nil, func(said string) { tripped = append(tripped, said) })
		var terminal bytes.Buffer
		s := Steps(IO{out: output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: &terminal, IsTerminal: true}, nil)})
		s.Step("installing "+token, detail)
		s.Stop()
		secret.Disarm()
		if len(tripped) != 0 {
			t.Errorf("%s tripped: %q", name, tripped)
		}
	}
}

func TestAStepThatEndedLeavesItsLineAtATerminalOnly(t *testing.T) {
	for name, terminal := range map[string]bool{"a terminal": true, "a pipe": false} {
		var stderr bytes.Buffer
		s := Steps(IO{out: output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: &stderr, IsTerminal: terminal}, map[string]string{"NO_COLOR": "1"})})
		s.Step("cloning", "")
		s.Done("cloning", nil)
		s.Step("booting", "")
		s.Done("booting", errors.New("no"))
		s.Stop()
		heard := stderr.String()
		if terminal != (strings.Contains(heard, " cloning") && strings.HasSuffix(heard, " <1s\n") && strings.Count(heard, "\n") == 1) {
			t.Errorf("%s heard %q", name, heard)
		}
		if !terminal && heard != "" {
			t.Errorf("%s heard %q, want nothing", name, heard)
		}
	}
}
