package runtimes_test

import (
	"testing"

	"spwn.sh/packages/runtimes"

	// Every built-in adapter registers via its init(); the defaults
	// package pulls them both in so these tests see the full set.
	_ "spwn.sh/packages/runtimes/defaults"
)

// TestAdapterRegistration verifies that blank-importing the defaults
// package (transitively, every built-in subpackage) lands both
// shipped runtimes in the Adapter registry. Failing here means a
// subpackage forgot its init() or the parent package accidentally
// filtered it out.
func TestAdapterRegistration(t *testing.T) {
	got := map[string]runtimes.Adapter{}
	for _, a := range runtimes.All() {
		if _, dup := got[a.Name]; dup {
			t.Errorf("duplicate adapter name: %s", a.Name)
		}
		got[a.Name] = a
	}

	for _, want := range []string{"claude-code", "codex"} {
		if _, ok := got[want]; !ok {
			t.Errorf("adapter %q not registered; got %v", want, runtimes.Names())
		}
	}
}

// TestAdapterIdentityFields locks in the invariant that every
// built-in adapter declares Name, CatalogRef, and DefaultProvider —
// the three bits of metadata the rest of the codebase reads to route
// work (dep-resolution, auth plumbing, runtime lookup).
func TestAdapterIdentityFields(t *testing.T) {
	for _, a := range runtimes.All() {
		t.Run(a.Name, func(t *testing.T) {
			if a.Name == "" {
				t.Error("empty Name")
			}
			if a.CatalogRef == "" {
				t.Errorf("%s: empty CatalogRef", a.Name)
			}
			if a.DefaultProvider == "" {
				t.Errorf("%s: empty DefaultProvider", a.Name)
			}
		})
	}
}

// TestAdapterShipInstallRecipe locks in that every shipped runtime
// has a Tool (install recipe). A future YAML-first or renderer-only
// runtime would drop this field, but today every adapter in defaults
// installs something in the image.
func TestAdapterShipInstallRecipe(t *testing.T) {
	for _, a := range runtimes.All() {
		t.Run(a.Name, func(t *testing.T) {
			if a.Tool == nil {
				t.Fatalf("%s: Tool is nil — built-in runtimes ship an install recipe", a.Name)
			}
			if a.Tool.Name() != a.CatalogRef {
				t.Errorf("%s: Tool.Name() = %q, want CatalogRef %q (dep-resolution needs them equal)", a.Name, a.Tool.Name(), a.CatalogRef)
			}
		})
	}
}

// TestAdapterRegisterSideEffects verifies that Adapter.Register
// (invoked at init time by each subpackage) side-registers the
// Spawn facet into the spawner registry and the Render facet into
// transpile's renderer registry. Without this, GetSpawner and
// transpile.Compile would return "not found" even though the
// Adapter is in All().
func TestAdapterRegisterSideEffects(t *testing.T) {
	for _, a := range runtimes.All() {
		if a.Spawn == nil {
			continue
		}
		t.Run(a.Name+"/spawner", func(t *testing.T) {
			s, err := runtimes.GetSpawner(a.Name)
			if err != nil {
				t.Fatalf("spawner registry missing %s: %v", a.Name, err)
			}
			if s.Name() != a.Name {
				t.Errorf("spawner Name() = %q, want %q", s.Name(), a.Name)
			}
		})
	}
}

// TestGet returns the adapter by short name, and reports ok=false
// for unknown names. The CLI relies on this for --runtime flag
// validation and friendly error messages.
func TestGet(t *testing.T) {
	if _, ok := runtimes.Get("nonexistent"); ok {
		t.Error("Get(nonexistent) returned ok=true")
	}
	a, ok := runtimes.Get("claude-code")
	if !ok {
		t.Fatal("Get(claude-code) returned ok=false")
	}
	if a.Name != "claude-code" {
		t.Errorf("Get returned adapter with Name=%q", a.Name)
	}
}

// TestNames returns every registered adapter name. Used by the CLI
// error-hint path ("unknown runtime; available: …"). Sort-order is
// insertion-order; the test just checks the content, not order.
func TestNames(t *testing.T) {
	names := runtimes.Names()
	if len(names) != len(runtimes.All()) {
		t.Errorf("Names() len=%d, All() len=%d", len(names), len(runtimes.All()))
	}
	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}
	for _, want := range []string{"claude-code", "codex"} {
		if !have[want] {
			t.Errorf("Names() missing %q: got %v", want, names)
		}
	}
}

// TestRegisterDefaults iterates every adapter with a Tool and
// registers it into the provided tool.Registry. This is the
// integration between the Adapter umbrella and the dep-resolver
// Registry. Failing here breaks `spwn build` and `spwn up`.
func TestRegisterDefaults(t *testing.T) {
	reg := &fakeToolRegistry{seen: map[string]bool{}}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		t.Fatalf("RegisterDefaults: %v", err)
	}
	// Every adapter with a Tool should land in the registry once.
	for _, a := range runtimes.All() {
		if a.Tool == nil {
			continue
		}
		if !reg.seen[a.Tool.Name()] {
			t.Errorf("RegisterDefaults did not register %s (%s)", a.Name, a.Tool.Name())
		}
	}
}
