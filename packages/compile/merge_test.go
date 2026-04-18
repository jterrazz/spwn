package compile

import "spwn.sh/packages/dependency/tool"

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

func parseJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, raw)
	}
	return m
}

func TestMergeRuntimeConfig_BaseOnly(t *testing.T) {
	got, err := MergeRuntimeConfig([]byte(`{"a":1,"b":"x"}`))
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	m := parseJSON(t, got)
	if m["a"].(float64) != 1 || m["b"] != "x" {
		t.Errorf("unexpected merged map: %v", m)
	}
}

func TestMergeRuntimeConfig_BasePlusOne(t *testing.T) {
	got, err := MergeRuntimeConfig(
		[]byte(`{"a":1}`),
		[]byte(`{"b":2}`),
	)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	m := parseJSON(t, got)
	if m["a"].(float64) != 1 || m["b"].(float64) != 2 {
		t.Errorf("unexpected merged map: %v", m)
	}
}

func TestMergeRuntimeConfig_LastWriteWins(t *testing.T) {
	got, err := MergeRuntimeConfig(
		[]byte(`{"k":"base"}`),
		[]byte(`{"k":"plugin-a"}`),
		[]byte(`{"k":"plugin-b"}`),
	)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	m := parseJSON(t, got)
	if m["k"] != "plugin-b" {
		t.Errorf("last-write-wins failed: got %v", m["k"])
	}
}

func TestMergeRuntimeConfig_NilInputs(t *testing.T) {
	got, err := MergeRuntimeConfig(nil, nil, []byte(`{"x":42}`), nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	m := parseJSON(t, got)
	if m["x"].(float64) != 42 {
		t.Errorf("x = %v, want 42", m["x"])
	}
}

func TestMergeRuntimeConfig_EmptyResult(t *testing.T) {
	got, err := MergeRuntimeConfig(nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if strings.TrimSpace(string(got)) != "{}" {
		t.Errorf("empty merge = %q, want {}", got)
	}
}

// fakeConfigPlugin is a minimal in-memory Tool with a dependency block
// used to drive CollectRuntimeConfigs tests — the architect's
// injection pipeline consumes the same helper.
type fakeConfigPlugin struct {
	name     string
	runtimes []string
	cfg      map[string]string
}

func (p *fakeConfigPlugin) Name() string           { return p.name }
func (p *fakeConfigPlugin) Kind() tool.Kind             { return tool.KindTool }
func (p *fakeConfigPlugin) Version() string        { return "0.0.0" }
func (p *fakeConfigPlugin) Dependencies() []string { return nil }
func (p *fakeConfigPlugin) Install() tool.InstallSpec   { return tool.InstallSpec{} }
func (p *fakeConfigPlugin) Verify() []string       { return nil }
func (p *fakeConfigPlugin) Skills() fs.FS          { return nil }
func (p *fakeConfigPlugin) Runtimes() []string     { return p.runtimes }
func (p *fakeConfigPlugin) Config(runtime string) []byte {
	if v, ok := p.cfg[runtime]; ok {
		return []byte(v)
	}
	return nil
}

func TestCollectRuntimeConfigs_FiltersByRuntime(t *testing.T) {
	resolved := []tool.Tool{
		&fakeConfigPlugin{
			name:     "spwn:a",
			runtimes: []string{"spwn:claude-code"},
			cfg:      map[string]string{"spwn:claude-code": `{"mcpServers":{"a":1}}`},
		},
		&fakeConfigPlugin{
			name:     "spwn:b",
			runtimes: []string{"spwn:codex"},
			cfg:      map[string]string{"spwn:codex": `{"b":true}`},
		},
		&fakeConfigPlugin{
			name:     "spwn:c",
			runtimes: []string{"spwn:claude-code"},
			cfg:      map[string]string{"spwn:claude-code": `{"marker":"ok"}`},
		},
	}

	cfgs := CollectRuntimeConfigs(resolved, "spwn:claude-code")
	if len(cfgs) != 2 {
		t.Fatalf("want 2 configs, got %d", len(cfgs))
	}

	merged, err := MergeRuntimeConfig([]byte(`{"hasCompletedOnboarding":true}`), cfgs...)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	m := parseJSON(t, merged)
	if m["hasCompletedOnboarding"] != true {
		t.Errorf("base key lost: %v", m)
	}
	if m["marker"] != "ok" {
		t.Errorf("dependency c marker missing: %v", m)
	}
	if _, ok := m["mcpServers"]; !ok {
		t.Errorf("dependency a mcpServers missing: %v", m)
	}
	// codex dependency must not leak in
	if _, ok := m["b"]; ok {
		t.Errorf("codex config leaked into claude-code merge: %v", m)
	}
}
