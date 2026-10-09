package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Tool is the gate's view of a catalog tool installed under
// ~/.spwn/gate/tools/<slug>/. The gate cares about three things:
//
//   - cookies: register a CookieProvider so the extension auto-syncs
//     the right session cookies for this tool.
//   - mcp.entry: the command to spawn at startup; gate reverse-proxies
//     /mcp/<slug>/* into the subprocess.
//   - dir: where the tool's files live, set as cwd for the subprocess.
//
// Everything the dependency package cares about (install commands,
// version, transitive deps, files for COPY-into-image) lives in the
// same tool.yaml but is the build pipeline's concern, not the gate's.
type Tool struct {
	Name string // slug, derived from the tool dir name (e.g. "x")
	Dir  string // absolute path to the tool's dir
	Spec ToolGateSpec
}

// ToolGateSpec is the `gate:` subsection of tool.yaml. Each
// installed catalog tool that wants to plug into the gate fills in
// this section; tools that only ship into world containers (apt
// packages, CLIs) leave it empty.
type ToolGateSpec struct {
	Cookies *ToolCookies `yaml:"cookies,omitempty"`
	MCP     *ToolMCP     `yaml:"mcp,omitempty"`
}

// ToolCookies declares which cookies the cookie-sync extension
// should push for this tool. Registered with CookieSync at startup;
// the extension picks it up on its next /sync/providers refresh.
type ToolCookies struct {
	Domains []string `yaml:"domains"`
	Cookies []string `yaml:"cookies"`
}

// ToolMCP declares the subprocess that serves the tool's MCP HTTP
// endpoint. Gate spawns it with these env vars set:
//
//   GATE_TOOL_NAME       — slug (= /mcp/<slug>/ route)
//   GATE_TOOL_PORT       — port assigned by gate, tool must listen on
//                          127.0.0.1:$GATE_TOOL_PORT/mcp/
//   GATE_BROWSER_URL     — http://127.0.0.1:9001 (sidecar)
//   GATE_CREDENTIALS_DIR — typically /credentials, where cookie + OAuth
//                          files live
type ToolMCP struct {
	Entry []string `yaml:"entry"`
}

// toolFileSchema is what we parse out of tool.yaml — only the gate-
// relevant fields. Everything else (install, verify, …) is handled
// by the dependency package elsewhere; we deliberately don't import
// it here to keep the gate self-contained.
type toolFileSchema struct {
	Name string       `yaml:"name"`
	Gate ToolGateSpec `yaml:"gate"`
}

// ToolsDir is the on-host root where the spwn CLI installs catalog
// tools that have a `gate:` section. Lives next to ~/.spwn/credentials/
// and ~/.spwn/gate/cookie-sync-secret. Bind-mounted into the gate
// container at /gate/tools/.
//
// Resolution: the gate (running inside Docker) sees /gate/tools/.
// The CLI (on the host) sees ~/.spwn/gate/tools/. The bind mount is
// configured by packages/gate/lifecycle.go.
const InContainerToolsDir = "/gate/tools"

// LoadTools scans `dir` for catalog tools (one subdir per tool with
// a tool.yaml). Returns the parsed list; subdirs without tool.yaml
// or without a `gate:` section are silently skipped (a project-local
// tool with only install steps is valid and not gate-relevant).
func LoadTools(dir string) ([]Tool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Tool
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		toolDir := filepath.Join(dir, e.Name())
		manifest := filepath.Join(toolDir, "tool.yaml")
		raw, err := os.ReadFile(manifest)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", manifest, err)
		}
		var sch toolFileSchema
		if err := yaml.Unmarshal(raw, &sch); err != nil {
			return nil, fmt.Errorf("parse %s: %w", manifest, err)
		}
		// Skip tools with no gate-relevant config.
		if sch.Gate.Cookies == nil && sch.Gate.MCP == nil {
			continue
		}
		out = append(out, Tool{
			Name: e.Name(),
			Dir:  toolDir,
			Spec: sch.Gate,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// CookieProvider derives the gate's CookieProvider record from a
// tool's gate.cookies declaration.
func (t Tool) CookieProvider() *CookieProvider {
	if t.Spec.Cookies == nil {
		return nil
	}
	return &CookieProvider{
		Name:    t.Name,
		Domains: t.Spec.Cookies.Domains,
		Cookies: t.Spec.Cookies.Cookies,
	}
}
