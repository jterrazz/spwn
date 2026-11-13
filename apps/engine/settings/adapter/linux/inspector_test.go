package linux_test

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/linux"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

// value: a Linux VM on the hypervisor's network, its resolver a stub and two providers.
func vm(c local.Cmd) (local.Output, error) {
	switch c.Name {
	case "hostname":
		return local.Output{Stdout: "192.168.64.5 fd00::5\n"}, nil
	case "cat":
		return local.Output{Stdout: "nameserver 127.0.0.1\nnameserver 10.0.2.3\nnameserver 10.0.2.3\nnameserver 1.1.1.1\n"}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func TestInspector(t *testing.T) {
	contracttest.Inspector(t, func(*testing.T) contracttest.InspectorSubject {
		return contracttest.InspectorSubject{Inspector: linux.New(local.Played(vm, "hostname", "cat")), Unreadable: catalog.FactFileVault, Present: catalog.FactLocalIP}
	})
	i := linux.New(local.Played(vm, "hostname", "cat"))
	for f, want := range map[catalog.Fact]string{catalog.FactLocalIP: "192.168.64.5", catalog.FactDNS: "10.0.2.3, 1.1.1.1"} {
		if got, err := i.Read(context.Background(), f); err != nil || got.Value != want {
			t.Errorf("Read(%s) = %+v, %v; want %q", f, got, err, want)
		}
	}
}
