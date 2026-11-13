package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func Holder(t *testing.T, sample catalog.Install, build Build) {
	// invariant: what every Holder owes the apps model: what an install
	// adds is held by the app it ran for, by that app's name; a removal
	// without the owner's word takes nothing station never recorded adding,
	// refused, and takes back what it recorded; on the word it takes the
	// owner's own; a key two apps' installs hold stays until the last of
	// them is removed.
	t.Helper()
	ctx := context.Background()
	key := sample.Key()
	in := []catalog.Install{sample}
	holderOf := func(t *testing.T, seed Seed) (contract.Installer, contract.Holder) {
		t.Helper()
		i := build(t, seed)
		h, ok := i.(contract.Holder)
		if !ok {
			t.Fatalf("%T is no Holder", i)
		}
		return i, h
	}
	listed := func(t *testing.T, i contract.Installer) bool {
		t.Helper()
		got, err := i.Inventory(ctx)
		if err != nil {
			t.Fatalf("inventory: %v", err)
		}
		_, ok := got[key]
		return ok
	}
	holds := func(t *testing.T, h contract.Holder, app string) bool {
		t.Helper()
		ok, err := h.Holds(ctx, app)
		if err != nil {
			t.Fatalf("holds %s: %v", app, err)
		}
		return ok
	}

	t.Run("the owner's own is taken on the owner's word alone", func(t *testing.T) {
		i, h := holderOf(t, Seed{Installed: map[string]string{key: "1.0.0"}})
		if err := h.InstallFor(ctx, contract.For{App: "one"}, in, nil); err != nil {
			t.Fatal(err)
		}
		if holds(t, h, "one") {
			t.Errorf("an install over the owner's %s recorded it the app's", key)
		}
		if err := h.RemoveFor(ctx, contract.For{App: "one"}, in, nil); err == nil || !listed(t, i) {
			t.Errorf("a removal without the owner's word gave %v, listed %v; want it refused, %s kept", err, listed(t, i), key)
		}
		if err := h.RemoveFor(ctx, contract.For{App: "one", Word: true}, in, nil); err != nil || listed(t, i) {
			t.Errorf("the owner's removal gave %v, listed %v; want %s gone", err, listed(t, i), key)
		}
	})

	t.Run("what an install adds is the app's, taken back without the word", func(t *testing.T) {
		i, h := holderOf(t, Seed{Installed: map[string]string{}})
		if err := h.InstallFor(ctx, contract.For{App: "one"}, in, nil); err != nil {
			t.Fatal(err)
		}
		if !holds(t, h, "one") || holds(t, h, "other") {
			t.Errorf("held by one %v, by other %v; want the app it ran for alone", holds(t, h, "one"), holds(t, h, "other"))
		}
		if err := h.RemoveFor(ctx, contract.For{App: "one"}, in, nil); err != nil || listed(t, i) || holds(t, h, "one") {
			t.Errorf("the removal gave %v, listed %v; want %s gone, its record with it", err, listed(t, i), key)
		}
	})

	t.Run("a key two apps hold stays until the last goes", func(t *testing.T) {
		i, h := holderOf(t, Seed{Installed: map[string]string{}})
		for _, app := range []string{"one", "other"} {
			if err := h.InstallFor(ctx, contract.For{App: app}, in, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := h.RemoveFor(ctx, contract.For{App: "one", Word: true}, in, nil); err != nil || !listed(t, i) {
			t.Errorf("removing one gave %v, listed %v; want %s kept for other", err, listed(t, i), key)
		}
		if err := h.RemoveFor(ctx, contract.For{App: "other"}, in, nil); err != nil || listed(t, i) {
			t.Errorf("removing other gave %v, listed %v; want %s gone", err, listed(t, i), key)
		}
	})
}
