package contracttest

import (
	"context"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

func Scene() []ports.Listener {
	// value: the scene every adapter's recorded inputs describe: node on every
	// IPv4 and IPv6 address at 3000, postgres on the loopback at 5432.
	return []ports.Listener{
		{Port: 3000, Bind: "0.0.0.0", Process: "node", PID: 100},
		{Port: 3000, Bind: "::", Process: "node", PID: 100},
		{Port: 5432, Bind: "127.0.0.1", Process: "postgres", PID: 101},
	}
}

func Listeners(t *testing.T, open func(t *testing.T) contract.Listeners) {
	t.Run("every listening address is read with the process that holds it", func(t *testing.T) {
		got, err := open(t).Read(context.Background())
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		want := Scene()
		order := func(a, b ports.Listener) int {
			if a.Port != b.Port {
				return a.Port - b.Port
			}
			if a.Bind < b.Bind {
				return -1
			}
			if a.Bind > b.Bind {
				return 1
			}
			return 0
		}
		slices.SortFunc(got, order)
		slices.SortFunc(want, order)
		if !slices.Equal(got, want) {
			t.Fatalf("Read =\n%+v\nwant\n%+v", got, want)
		}
	})
}
