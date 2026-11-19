package wire

import (
	"context"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: the one place the engine asks macOS for Local Network: a
// datagram into the local network (an mDNS query to its group), which
// macOS refuses while the Mac has not approved station, and at the first
// refusal asks the owner on the Mac's screen. Off a Mac it reads nothing.
type localNetworkRead struct {
	sockets local.Sockets
	goos    string
	to      string
	payload []byte
}

func mdnsQuery() []byte {
	// value: an mDNS query for _station._tcp.local's PTR, one question:
	// a header of one question, the name's labels, then type 12 class 1.
	return []byte("\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x08_station\x04_tcp\x05local\x00\x00\x0c\x00\x01")
}

func (r localNetworkRead) read(ctx context.Context) model.LocalNetwork {
	err := r.sockets.SendUDP(ctx, r.to, r.payload)
	switch {
	case err == nil:
		return model.LocalNetworkApproved
	case local.BlockedByLocalNetwork(r.goos, err):
		return model.LocalNetworkNotApproved
	}
	return ""
}

func (m machine) localNetwork() func(ctx context.Context) model.LocalNetwork {
	if m.world != nil {
		return m.world.localNetwork()
	}
	if m.sys.Env.GOOS() != "darwin" {
		return nil
	}
	return localNetworkRead{sockets: m.sys.Sockets, goos: m.sys.Env.GOOS(), to: "224.0.0.251:5353", payload: mdnsQuery()}.read
}
