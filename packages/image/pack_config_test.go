package image

import (
	"io/fs"
	"testing"
)

// baseTool is a minimal Tool used to exercise the pack helpers
// against a plain package (no runtime-config: block). Runtimes() returns nil
// so PluginConfig short-circuits.
type baseTool struct{ name string }

func (t *baseTool) Name() string            { return t.name }
func (t *baseTool) Kind() Kind              { return KindTool }
func (t *baseTool) Version() string         { return "0.0.0" }
func (t *baseTool) Dependencies() []string  { return nil }
func (t *baseTool) Install() InstallSpec    { return InstallSpec{} }
func (t *baseTool) Verify() []string        { return nil }
func (t *baseTool) Skills() fs.FS           { return nil }
func (t *baseTool) Runtimes() []string      { return nil }
func (t *baseTool) Config(string) []byte    { return nil }

// pluginTool targets specific runtimes and returns per-runtime config.
// Used to verify PluginConfig's allowlist gating.
type pluginTool struct {
	runtimes []string
	config   map[string][]byte
}

func (t *pluginTool) Name() string                 { return "@spwn/fake" }
func (t *pluginTool) Kind() Kind                   { return KindTool }
func (t *pluginTool) Version() string              { return "0.0.0" }
func (t *pluginTool) Dependencies() []string       { return nil }
func (t *pluginTool) Install() InstallSpec         { return InstallSpec{} }
func (t *pluginTool) Verify() []string             { return nil }
func (t *pluginTool) Skills() fs.FS                { return nil }
func (t *pluginTool) Runtimes() []string           { return t.runtimes }
func (t *pluginTool) Config(runtime string) []byte { return t.config[runtime] }

func TestPluginRuntimes_PlainTool(t *testing.T) {
	// A package with no runtime-config: block returns nil from both helpers.
	if got := PluginRuntimes(&baseTool{name: "@spwn/plain"}); got != nil {
		t.Errorf("PluginRuntimes(plain) = %v, want nil", got)
	}
	if got := PluginConfig(&baseTool{name: "@spwn/plain"}, "@spwn/claude-code"); got != nil {
		t.Errorf("PluginConfig(plain) = %v, want nil", got)
	}
}

func TestPluginConfig_RuntimeGate(t *testing.T) {
	marker := []byte(`{"hello":"world"}`)
	p := &pluginTool{
		runtimes: []string{"@spwn/claude-code"},
		config:   map[string][]byte{"@spwn/claude-code": marker, "@spwn/codex": marker},
	}
	// Matching runtime → config flows through.
	if got := PluginConfig(p, "@spwn/claude-code"); string(got) != string(marker) {
		t.Errorf("PluginConfig(claude-code) = %q, want %q", got, marker)
	}
	// Non-declared runtime → gated to nil even if Config would return bytes.
	if got := PluginConfig(p, "@spwn/codex"); got != nil {
		t.Errorf("PluginConfig(codex) = %q, want nil (gated)", got)
	}
}

func TestPluginConfig_EmptyRuntimesIsNotAPlugin(t *testing.T) {
	// Under the unified Tool interface, empty Runtimes() means "not
	// a pack" — every runtime gets nil back regardless of what
	// Config would return. Previously this was "runtime-agnostic
	// plugin" but that concept is gone: packs must opt in to a
	// specific runtime list.
	p := &pluginTool{
		runtimes: nil,
		config:   map[string][]byte{"@spwn/claude-code": []byte("ok")},
	}
	if got := PluginConfig(p, "@spwn/claude-code"); got != nil {
		t.Errorf("PluginConfig(no-runtimes) = %q, want nil", got)
	}
}
