package plugins

import (
	"testing"

	ib "spwn.sh/packages/image"
)

// A standalone sanity test: RegisterDefaults into a fresh registry
// should succeed and expose every plugin by name.
func TestRegisterDefaults(t *testing.T) {
	reg := ib.NewRegistry()
	if err := RegisterDefaults(reg); err != nil {
		t.Fatalf("RegisterDefaults: %v", err)
	}
	for _, p := range All {
		if got := reg.Get(p.Name()); got == nil {
			t.Errorf("plugin %q not found in registry after RegisterDefaults", p.Name())
		}
	}
}

// Plugins must advertise runtimes via the optional Plugin interface
// (or opt out explicitly). A plugin with an empty Runtimes() slice is
// runtime-agnostic — acceptable but unusual — so the test only checks
// that the optional methods don't panic.
func TestAllPlugins_RuntimesCallable(t *testing.T) {
	for _, p := range All {
		_ = ib.PluginRuntimes(p) // must not panic
		for _, r := range ib.PluginRuntimes(p) {
			_ = ib.PluginConfig(p, r) // must not panic
		}
	}
}
