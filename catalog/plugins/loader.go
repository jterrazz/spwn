package plugins

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/image/toolyaml"
)

// yamlPluginsFS embeds every YAML-defined plugin. Each plugin lives
// at catalog/plugins/<name>/spwn-tool.yaml with optional sibling
// skills/ and config/ directories. The plugin: section in the
// manifest scopes which runtimes receive the injected config at
// spawn time.
//
//go:embed all:*
var yamlPluginsFS embed.FS

// loadYAMLPlugins mirrors loadYAMLTools in the tools package —
// walks the embedded tree and parses every spwn-tool.yaml.
// Directories without a manifest are ignored so transitional
// Go-based plugins can coexist.
func loadYAMLPlugins() ([]ib.Tool, error) {
	entries, err := fs.ReadDir(yamlPluginsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded plugins: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		manifestPath := path.Join(name, toolyaml.Manifest)
		if _, err := fs.Stat(yamlPluginsFS, manifestPath); err != nil {
			continue // legacy Go plugin
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var out []ib.Tool
	for _, name := range names {
		tool, err := toolyaml.Parse(
			toolyaml.EmbedResolver{FS: yamlPluginsFS, Root: name},
			toolyaml.ParseOptions{
				DefaultName:    "@spwn/" + strings.ReplaceAll(name, "_", "-"),
				DefaultVersion: "latest",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load plugin %s: %w", name, err)
		}
		out = append(out, tool)
	}
	return out, nil
}
