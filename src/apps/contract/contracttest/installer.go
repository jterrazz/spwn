package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
)

type Seed struct {
	Absent    bool
	Installed map[string]string
	Fails     bool
}

type Build func(t *testing.T, seed Seed) contract.Installer

// value: the app a Holder's verbs run for here, on the owner's word, as
// the apps model runs an install and a removal asked by name.
var owner = contract.For{App: "sample", Word: true}

func install(ctx context.Context, i contract.Installer, in catalog.Install) error {
	if h, ok := i.(contract.Holder); ok {
		return h.InstallFor(ctx, owner, []catalog.Install{in}, nil)
	}
	return i.Install(ctx, []catalog.Install{in}, nil)
}

func remove(ctx context.Context, i contract.Installer, in catalog.Install) error {
	if h, ok := i.(contract.Holder); ok {
		return h.RemoveFor(ctx, owner, []catalog.Install{in}, nil)
	}
	return i.Remove(ctx, []catalog.Install{in}, nil)
}

func Installer(t *testing.T, sample catalog.Install, build Build) {
	// invariant: what every Installer owes the apps model, whatever runs it:
	// what it holds is listed, what it installs or removes shows in its next
	// list, a failed run is an error, a missing manager is not ready, and a
	// verb it cannot do is refused, never faked. A Holder installs and
	// removes through its own verbs, for one app, on the owner's word.
	t.Helper()
	ctx := context.Background()
	key := sample.Key()
	held := func() Seed { return Seed{Installed: map[string]string{key: "1.0.0"}} }
	listed := func(t *testing.T, i contract.Installer) (map[string]string, bool) {
		t.Helper()
		got, err := i.Inventory(ctx)
		if errors.Is(err, contract.ErrUnlisted) {
			return nil, false
		}
		if err != nil {
			t.Fatalf("inventory: %v", err)
		}
		return got, true
	}

	t.Run("its method is the one it is registered under", func(t *testing.T) {
		if got := build(t, Seed{}).Method(); got != sample.Method() {
			t.Errorf("an installer of %s says it is %s", sample.Method(), got)
		}
	})

	t.Run("an app it holds is listed by its key", func(t *testing.T) {
		got, ok := listed(t, build(t, held()))
		if !ok {
			t.Skip("its method keeps no list")
		}
		if _, in := got[key]; !in {
			t.Errorf("%s is held, the inventory lists %v", key, got)
		}
	})

	t.Run("what it installs is listed after", func(t *testing.T) {
		i := build(t, Seed{Installed: map[string]string{}})
		err := install(ctx, i, sample)
		if !i.Can().Install {
			if err == nil {
				t.Errorf("an installer that cannot install said it installed %s", key)
			}
			return
		}
		if err != nil {
			t.Fatalf("install: %v", err)
		}
		if got, ok := listed(t, i); ok {
			if _, in := got[key]; !in {
				t.Errorf("%s installed, the inventory lists %v", key, got)
			}
		}
	})

	t.Run("what it removes is no longer listed", func(t *testing.T) {
		i := build(t, held())
		err := remove(ctx, i, sample)
		if !i.Can().Remove {
			if err == nil {
				t.Errorf("an installer that cannot remove said it removed %s", key)
			}
			return
		}
		if err != nil {
			t.Fatalf("remove: %v", err)
		}
		if got, ok := listed(t, i); ok {
			if _, in := got[key]; in {
				t.Errorf("%s removed, the inventory still lists it", key)
			}
		}
	})

	t.Run("an installed app updates", func(t *testing.T) {
		i := build(t, held())
		err := i.Update(ctx, []catalog.Install{sample}, nil)
		switch {
		case !i.Can().Update && err == nil:
			t.Errorf("an installer that cannot update said it updated %s", key)
		case i.Can().Update && err != nil:
			t.Errorf("update: %v", err)
		}
	})

	t.Run("a failed run is an error, never a silent success", func(t *testing.T) {
		i := build(t, Seed{Installed: map[string]string{}, Fails: true})
		if !i.Can().Install {
			t.Skip("it installs nothing")
		}
		if err := install(ctx, i, sample); err == nil {
			t.Errorf("a failing install of %s returned no error", key)
		}
	})

	t.Run("a missing manager is not ready", func(t *testing.T) {
		i := build(t, Seed{Absent: true})
		if i.Manager().Name == "" {
			t.Skip("its method runs no manager")
		}
		if err := i.Ready(ctx); err == nil {
			t.Errorf("%s is absent, the installer says it is ready", i.Manager().Name)
		}
		if _, err := i.Inventory(ctx); err != nil && !errors.Is(err, contract.ErrMissing) {
			t.Errorf("an absent manager's inventory failed as %v, not as missing", err)
		}
	})

	t.Run("a bundled method spells its line", func(t *testing.T) {
		lined, ok := build(t, Seed{}).(contract.Lined)
		if !ok {
			t.Skip("its installs ride no bundle")
		}
		if lined.Line(sample) == "" {
			t.Errorf("no line for %s", key)
		}
	})
}
