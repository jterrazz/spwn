package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Bundled struct {
	// value: a bundle and an installer of one method whose lines it runs,
	// over the same seeded state.
	Bundler   contract.Bundler
	Installer contract.Installer
}

func Bundler(t *testing.T, sample catalog.Install, build func(t *testing.T, seed Seed) Bundled) {
	// invariant: what every Bundler owes the apps model: its run is named in a
	// verdict, a line it runs is installed and listed after, an update runs
	// over what is held, and a failed run is an error.
	t.Helper()
	ctx := context.Background()
	key := sample.Key()
	line := func(t *testing.T, b Bundled) string {
		t.Helper()
		lined, ok := b.Installer.(contract.Lined)
		if !ok {
			t.Fatalf("an installer of %s rides no bundle", sample.Method())
		}
		return lined.Line(sample)
	}

	t.Run("its run has a name a verdict can say", func(t *testing.T) {
		if build(t, Seed{}).Bundler.Name() == "" {
			t.Error("the bundle has no name")
		}
	})

	t.Run("a line it runs is listed after", func(t *testing.T) {
		b := build(t, Seed{Installed: map[string]string{}})
		if err := b.Bundler.Bundle(ctx, []string{line(t, b)}, false, nil); err != nil {
			t.Fatalf("bundle: %v", err)
		}
		got, err := b.Installer.Inventory(ctx)
		if err != nil {
			t.Fatalf("inventory: %v", err)
		}
		if _, in := got[key]; !in {
			t.Errorf("%s was bundled, the inventory lists %v", key, got)
		}
	})

	t.Run("an update runs over what is held", func(t *testing.T) {
		b := build(t, Seed{Installed: map[string]string{key: "1.0.0"}})
		if err := b.Bundler.Bundle(ctx, []string{line(t, b)}, true, nil); err != nil {
			t.Errorf("update bundle: %v", err)
		}
	})

	t.Run("a failed run is an error", func(t *testing.T) {
		b := build(t, Seed{Fails: true, Installed: map[string]string{}})
		if err := b.Bundler.Bundle(ctx, []string{line(t, b)}, false, nil); err == nil {
			t.Error("a failing bundle reported success")
		}
	})
}
