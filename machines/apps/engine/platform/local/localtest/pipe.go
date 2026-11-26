package localtest

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func Pipe(t testing.TB) (server, client local.NetConn) {
	// value: local.Pair's two ends, closed with the test: a protocol's
	// server on one end and its client on the other.
	t.Helper()
	server, client, err := local.Pair()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })
	return server, client
}
