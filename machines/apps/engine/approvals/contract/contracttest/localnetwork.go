package contracttest

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/approvals/contract"
)

type Open func(t *testing.T, answer approvals.State) contract.LocalNetwork

func LocalNetwork(t *testing.T, open Open) {
	// value: open is a probe whose Mac answers as given: approved,
	// refused, or unknown.
	for _, answer := range []approvals.State{approvals.Approved, approvals.Refused, approvals.Unknown} {
		t.Run("a Mac that answers "+string(answer)+" reads "+string(answer), func(t *testing.T) {
			if got := open(t, answer).Read(t.Context()); got != answer {
				t.Errorf("read %q, want %q", got, answer)
			}
		})
	}
}
