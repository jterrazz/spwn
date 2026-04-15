package runtimes

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

// yamlRuntimesFS embeds every YAML-defined runtime. Each runtime
// lives at catalog/runtimes/<name>/spwn-tool.yaml with optional
// sibling skills/ and files/ directories.
//
//go:embed all:*
var yamlRuntimesFS embed.FS

// loadYAMLRuntimes mirrors loadYAMLTools in the tools package —
// walks the embedded tree and parses every spwn-tool.yaml.
// Directories without a manifest are ignored so transitional
// Go-based runtimes can coexist.
func loadYAMLRuntimes() ([]ib.Tool, error) {
	entries, err := fs.ReadDir(yamlRuntimesFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded runtimes: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		manifestPath := path.Join(name, toolyaml.Manifest)
		if _, err := fs.Stat(yamlRuntimesFS, manifestPath); err != nil {
			continue // legacy Go runtime
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var out []ib.Tool
	for _, name := range names {
		tool, err := toolyaml.Parse(
			toolyaml.EmbedResolver{FS: yamlRuntimesFS, Root: name},
			toolyaml.ParseOptions{
				DefaultName:    "@spwn/" + strings.ReplaceAll(name, "_", "-"),
				DefaultVersion: "latest",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load runtime %s: %w", name, err)
		}
		out = append(out, tool)
	}
	return out, nil
}
