package docker

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func TestAContainerNamesTheOnePortItPublishes(t *testing.T) {
	got := ParsePS([]byte(`{"Names":"web-1","Ports":"0.0.0.0:3000->80/tcp, [::]:3000->80/tcp, 5432/tcp"}` + "\n"))
	if len(got) != 1 || got[0] != (ports.Announced{Port: 3000, IP: "0.0.0.0", Process: "web-1", Source: ports.ByContainer}) {
		t.Fatalf("published %+v", got)
	}
	merged := ports.MergeAnnounced([]ports.Announced{{Port: 3000, IP: "0.0.0.0", Process: "docker-proxy", Source: ports.BySocket}}, got)
	if merged[0].Process != "web-1" {
		t.Fatalf("the container does not win its port: %+v", merged)
	}
}
