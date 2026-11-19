package wire

import (
	"context"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

type sending struct {
	local.Sockets
	err  error
	sent []byte
}

func (s *sending) SendUDP(_ context.Context, _ string, payload []byte) error {
	s.sent = payload
	return s.err
}

func TestAMacReadsItsApprovalFromWhetherItsQueryLeaves(t *testing.T) {
	for _, c := range []struct {
		name string
		goos string
		err  error
		want model.LocalNetwork
	}{
		{"sent on a Mac", "darwin", nil, model.LocalNetworkApproved},
		{"route refused on a Mac", "darwin", localtest.Unreachable(), model.LocalNetworkNotApproved},
		{"another refusal on a Mac", "darwin", localtest.Refused(), ""},
		{"route refused off a Mac", "linux", localtest.Unreachable(), ""},
	} {
		s := &sending{err: c.err}
		got := localNetworkRead{sockets: s, goos: c.goos, to: "224.0.0.251:5353", payload: mdnsQuery()}.read(t.Context())
		if got != c.want {
			t.Errorf("%s: read %q, want %q", c.name, got, c.want)
		}
		if len(s.sent) != 37 || s.sent[5] != 1 || s.sent[12] != 8 {
			t.Errorf("%s: sent %q, want one question for _station._tcp.local", c.name, s.sent)
		}
	}
}
