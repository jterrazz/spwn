package wire

import (
	"context"
	"errors"
	"testing"

	appsfake "github.com/jterrazz/studio-station/src/apps/adapter/fake"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
	"github.com/jterrazz/studio-station/src/world"
)

func TestEveryFakeInstallerAnswersAsItsRealAdapterDoes(t *testing.T) {
	// invariant: for every method of the registry, the fake a world holds
	// names the manager, the abilities, the lack of a list, the lack of a
	// currency and the readiness the real adapter has, on a machine where no
	// program is found: a story over the fakes reads an app as a machine does.
	ctx := context.Background()
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	system.Run = local.Played(func(local.Cmd) (local.Output, error) { return local.Output{ExitCode: 1}, nil })
	adapters, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	w := &world.World{ThisMac: "here", Managers: map[catalog.Method]world.Named{}}
	w.Mac("here").Apps = map[catalog.Method]*world.Manager{}
	for method, adapter := range adapters {
		w.Managers[method] = appsfake.Named(adapter)
		w.Mac("here").Apps[method] = &world.Manager{Absent: true, Installed: map[string]string{}}
	}
	for method, adapter := range adapters {
		_, lined := adapter.(contract.Lined)
		fake := appsfake.InstallerOf(w, "here", method, lined)
		if fake.Manager() != adapter.Manager() || fake.Can() != adapter.Can() {
			t.Errorf("%s: the fake is %+v %+v, the adapter %+v %+v", method, fake.Manager(), fake.Can(), adapter.Manager(), adapter.Can())
		}
		if method.KeepsList() != !errors.Is(inventoryErr(ctx, adapter), contract.ErrUnlisted) || method.KeepsList() != !errors.Is(inventoryErr(ctx, fake), contract.ErrUnlisted) {
			t.Errorf("%s: the catalog says it keeps a list: %v; the adapter and the fake answer %v, %v", method, method.KeepsList(), inventoryErr(ctx, adapter), inventoryErr(ctx, fake))
		}
		if errors.Is(adapter.Ready(ctx), contract.ErrMissing) != errors.Is(fake.Ready(ctx), contract.ErrMissing) {
			t.Errorf("%s: with no program found, the adapter is ready %v, the fake %v", method, adapter.Ready(ctx), fake.Ready(ctx))
		}
		keys := []string{"sample"}
		seen, err := adapter.Outdated(ctx, keys)
		if kept := seen != nil || err != nil; kept != method.ReadsCurrency() {
			t.Errorf("%s: the catalog says it reads a currency: %v; the adapter answered %v, %v", method, method.ReadsCurrency(), seen, err)
		}
		if got, _ := fake.Outdated(ctx, keys); (got != nil) != method.ReadsCurrency() {
			t.Errorf("%s: the catalog says it reads a currency: %v; the fake answered %v", method, method.ReadsCurrency(), got)
		}
	}
}

func inventoryErr(ctx context.Context, i contract.Installer) error {
	_, err := i.Inventory(ctx)
	return err
}
