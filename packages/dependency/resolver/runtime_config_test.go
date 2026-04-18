package resolver

import (
	"io/fs"
	"testing"

	"spwn.sh/packages/dependency/tool"
)

// baseTool is a minimal Tool used to exercise the dependency helpers
// against a plain package (no runtime-config: block). Runtimes() returns nil
// so tool.PluginConfig short-circuits.
type baseTool struct{ name string }

func (t *baseTool) Name() string            { return t.name }
func (t *baseTool) Kind() tool.Kind              { return tool.KindTool }
func (t *baseTool) Version() string         { return "0.0.0" }
func (t *baseTool) Dependencies() []string  { return nil }
func (t *baseTool) Install() tool.InstallSpec    { return tool.InstallSpec{} }
func (t *baseTool) Verify() []string        { return nil }
func (t *baseTool) Skills() fs.FS           { return nil }
func (t *baseTool) Runtimes() []string      { return nil }
func (t *baseTool) Config(string) []byte    { return nil }

// packTool targets specific runtimes and returns per-runtime config.
// Used to verify tool.PluginConfig's allowlist gating.
type packTool struct {
	runtimes []string
	config   map[string][]byte
}

func (t *packTool) Name() string                 { return "spwn:fake" }
func (t *packTool) Kind() tool.Kind                   { return tool.KindTool }
func (t *packTool) Version() string              { return "0.0.0" }
func (t *packTool) Dependencies() []string       { return nil }
func (t *packTool) Install() tool.InstallSpec         { return tool.InstallSpec{} }
func (t *packTool) Verify() []string             { return nil }
func (t *packTool) Skills() fs.FS                { return nil }
func (t *packTool) Runtimes() []string           { return t.runtimes }
func (t *packTool) Config(runtime string) []byte { return t.config[runtime] }

func TestPackRuntimes_PlainTool(t *testing.T) {
	// A package with no runtime-config: block returns nil from both helpers.
	if got := tool.PluginRuntimes(&baseTool{name: "spwn:plain"}); got != nil {
		t.Errorf("tool.PluginRuntimes(plain) = %v, want nil", got)
	}
	if got := tool.PluginConfig(&baseTool{name: "spwn:plain"}, "spwn:claude-code"); got != nil {
		t.Errorf("tool.PluginConfig(plain) = %v, want nil", got)
	}
}

func TestPackConfig_RuntimeGate(t *testing.T) {
	marker := []byte(`{"hello":"world"}`)
	p := &packTool{
		runtimes: []string{"spwn:claude-code"},
		config:   map[string][]byte{"spwn:claude-code": marker, "spwn:codex": marker},
	}
	// Matching runtime → config flows through.
	if got := tool.PluginConfig(p, "spwn:claude-code"); string(got) != string(marker) {
		t.Errorf("tool.PluginConfig(claude-code) = %q, want %q", got, marker)
	}
	// Non-declared runtime → gated to nil even if Config would return bytes.
	if got := tool.PluginConfig(p, "spwn:codex"); got != nil {
		t.Errorf("tool.PluginConfig(codex) = %q, want nil (gated)", got)
	}
}

func TestPackConfig_EmptyRuntimesIsNotAPack(t *testing.T) {
	// Under the unified Tool interface, empty Runtimes() means "not
	// a dependency" — every runtime gets nil back regardless of what
	// Config would return. Previously this was "runtime-agnostic
	// plugin" but that concept is gone: dependencies must opt in to a
	// specific runtime list.
	p := &packTool{
		runtimes: nil,
		config:   map[string][]byte{"spwn:claude-code": []byte("ok")},
	}
	if got := tool.PluginConfig(p, "spwn:claude-code"); got != nil {
		t.Errorf("tool.PluginConfig(no-runtimes) = %q, want nil", got)
	}
}
