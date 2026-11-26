package macos

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the mDNS group a probe's datagram is sent to.
const mdnsGroup = "224.0.0.251:5353"

type LocalNetwork struct {
	// invariant: the one place the engine asks macOS for Local Network: a
	// datagram into the local network (an mDNS query to its group), which
	// macOS refuses while the Mac has not approved Spwn, and at the first
	// refusal asks the owner on the Mac's screen.
	Sockets local.Sockets
	GOOS    string
}

func mdnsQuery() []byte {
	// value: an mDNS query for _station._tcp.local's PTR, one question:
	// a header of one question, the name's labels, then type 12 class 1.
	return []byte("\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x08_station\x04_tcp\x05local\x00\x00\x0c\x00\x01")
}

func (r LocalNetwork) Read(ctx context.Context) approvals.State {
	err := r.Sockets.SendUDP(ctx, mdnsGroup, mdnsQuery())
	switch {
	case err == nil:
		return approvals.Approved
	case local.BlockedByLocalNetwork(r.GOOS, err):
		return approvals.Refused
	}
	return approvals.Unknown
}
