package wire

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type askedNet struct {
	local.Net
	urls []string
}

func (n *askedNet) Get(_ context.Context, url string, _ map[string]string) (status int, body []byte, err error) {
	n.urls = append(n.urls, url)
	return 404, nil, nil
}

func TestTheImageRegistryIsAskedForTheOwnerStationPublishesUnder(t *testing.T) {
	net := &askedNet{}
	sys := local.New(nil, nil, io.Discard, io.Discard)
	sys.Net = net

	_, _ = (machine{sys: sys}).imageSource("runner").Latest(t.Context(), "base-ubuntu")

	if len(net.urls) == 0 || !strings.Contains(net.urls[0], "repository:jterrazz/station-base-ubuntu") {
		t.Fatalf("the registry was asked %v, want the jterrazz/station-base-ubuntu repository", net.urls)
	}
}
