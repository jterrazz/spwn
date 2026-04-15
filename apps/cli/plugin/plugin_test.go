package plugin

import (
	"bytes"
	"strings"
	"testing"

	plugins "spwn.sh/catalog/plugins"
	runtimes "spwn.sh/catalog/runtimes"
	tools "spwn.sh/catalog/tools"
	ib "spwn.sh/packages/image"
)

func TestPluginLsCmd_RunsSuccessfully(t *testing.T) {
	// ls should always run without error — even when the registered
	// plugin list is empty, it prints a friendly message.
	var out bytes.Buffer
	lsCmd.SetOut(&out)
	lsCmd.SetErr(&out)
	if err := lsCmd.RunE(lsCmd, nil); err != nil {
		t.Fatalf("ls RunE: %v", err)
	}
	got := out.String()
	if got == "" {
		t.Error("ls produced no output")
	}
}

// TestMempalace_ResolvesAgainstFullRegistry verifies that the full
// catalog (tools + runtimes + plugins) registers without collision
// and that @spwn/mempalace's dependency chain resolves end-to-end.
// This covers the integration point where plugins reach into the
// tool catalog for their dependencies.
func TestMempalace_ResolvesAgainstFullRegistry(t *testing.T) {
	reg := ib.NewRegistry()
	if err := tools.RegisterDefaults(reg); err != nil {
		t.Fatalf("tools.RegisterDefaults: %v", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		t.Fatalf("runtimes.RegisterDefaults: %v", err)
	}
	if err := plugins.RegisterDefaults(reg); err != nil {
		t.Fatalf("plugins.RegisterDefaults: %v", err)
	}

	resolved, err := reg.Resolve([]string{"@spwn/unix", "@spwn/claude-code", "@spwn/mempalace"})
	if err != nil {
		t.Fatalf("resolve mempalace chain: %v", err)
	}

	names := map[string]bool{}
	for _, t := range resolved {
		names[t.Name()] = true
	}
	for _, want := range []string{"@spwn/unix", "@spwn/python", "@spwn/mempalace", "@spwn/claude-code"} {
		if !names[want] {
			t.Errorf("resolved chain missing %q; got %v", want, names)
		}
	}
}

func TestPluginShowCmd_UnknownReturnsError(t *testing.T) {
	var out bytes.Buffer
	showCmd.SetOut(&out)
	showCmd.SetErr(&out)
	err := showCmd.RunE(showCmd, []string{"@spwn/does-not-exist"})
	if err == nil {
		t.Fatal("show on unknown plugin should error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}
