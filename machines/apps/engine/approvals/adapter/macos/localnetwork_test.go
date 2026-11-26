package macos

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/approvals/contract"
	"github.com/jterrazz/studio-station/apps/engine/approvals/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

type sending struct {
	local.Sockets
	err  error
	to   string
	sent []byte
}

func (s *sending) SendUDP(_ context.Context, to string, payload []byte) error {
	s.to, s.sent = to, payload
	return s.err
}

func TestTheMacOSProbeKeepsTheContract(t *testing.T) {
	contracttest.LocalNetwork(t, func(_ *testing.T, answer approvals.State) contract.LocalNetwork {
		var err error
		switch answer {
		case approvals.Refused:
			err = localtest.Unreachable()
		case approvals.Unknown:
			err = localtest.Refused()
		case approvals.Approved, approvals.Checking:
		}
		return LocalNetwork{Sockets: &sending{err: err}, GOOS: "darwin"}
	})
}

func TestAMacReadsItsApprovalFromWhetherItsQueryLeaves(t *testing.T) {
	for _, c := range []struct {
		name string
		goos string
		err  error
		want approvals.State
	}{
		{"sent on a Mac", "darwin", nil, approvals.Approved},
		{"route refused on a Mac", "darwin", localtest.Unreachable(), approvals.Refused},
		{"another refusal on a Mac", "darwin", localtest.Refused(), approvals.Unknown},
		{"route refused off a Mac", "linux", localtest.Unreachable(), approvals.Unknown},
	} {
		s := &sending{err: c.err}
		got := LocalNetwork{Sockets: s, GOOS: c.goos}.Read(t.Context())
		if got != c.want {
			t.Errorf("%s: read %q, want %q", c.name, got, c.want)
		}
		if s.to != "224.0.0.251:5353" || len(s.sent) != 37 || s.sent[5] != 1 || s.sent[12] != 8 {
			t.Errorf("%s: sent %q to %s, want one question for _station._tcp.local to the mDNS group", c.name, s.sent, s.to)
		}
	}
}
