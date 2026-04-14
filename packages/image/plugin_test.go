package image

import (
	"io/fs"
	"testing"
)

// baseTool is a minimal Tool used to exercise the plugin helpers against
// a plain (non-Plugin) implementation.
type baseTool struct{ name string }

func (t *baseTool) Name() string           { return t.name }
func (t *baseTool) Kind() Kind             { return KindTool }
func (t *baseTool) Version() string        { return "0.0.0" }
func (t *baseTool) Dependencies() []string { return nil }
func (t *baseTool) Install() InstallSpec   { return InstallSpec{} }
func (t *baseTool) Verify() []string       { return nil }
func (t *baseTool) Skills() fs.FS          { return nil }

// pluginTool embeds PluginBase and overrides it to target a runtime.
type pluginTool struct {
	PluginBase
	runtimes []string
	config   map[string][]byte
}

func (t *pluginTool) Name() string                  { return "@spwn/fake" }
func (t *pluginTool) Kind() Kind                    { return KindTool }
func (t *pluginTool) Version() string               { return "0.0.0" }
func (t *pluginTool) Dependencies() []string        { return nil }
func (t *pluginTool) Install() InstallSpec          { return InstallSpec{} }
func (t *pluginTool) Verify() []string              { return nil }
func (t *pluginTool) Skills() fs.FS                 { return nil }
func (t *pluginTool) Runtimes() []string            { return t.runtimes }
func (t *pluginTool) Config(runtime string) []byte  { return t.config[runtime] }

func TestPluginBase_Defaults(t *testing.T) {
	var pb PluginBase
	if got := pb.Runtimes(); got != nil {
		t.Errorf("Runtimes() = %v, want nil", got)
	}
	if got := pb.Config("@spwn/claude-code"); got != nil {
		t.Errorf("Config() = %v, want nil", got)
	}
}

func TestPluginRuntimes_PlainTool(t *testing.T) {
	// A plain Tool is not a Plugin: helpers should return zero values.
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

func TestPluginConfig_Agnostic(t *testing.T) {
	// An agnostic plugin (empty Runtimes) passes every runtime through.
	p := &pluginTool{
		runtimes: nil,
		config:   map[string][]byte{"@spwn/claude-code": []byte("ok")},
	}
	if got := PluginConfig(p, "@spwn/claude-code"); string(got) != "ok" {
		t.Errorf("PluginConfig agnostic = %q, want %q", got, "ok")
	}
}
