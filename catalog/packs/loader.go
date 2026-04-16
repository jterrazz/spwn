package packs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/deps"
)

// yamlToolsFS embeds every YAML-defined package. Each package lives
// at catalog/packs/<name>/spwn.yaml with optional sibling
// skills/, files/, and config/ directories. The embed is rooted at
// the catalog/packs/ directory so the walk sees
// <name>/spwn.yaml entries directly.
//
// Adding a new YAML package? Drop the directory in and re-build —
// the loader picks it up automatically. No registration list to
// maintain.
//
//go:embed all:*
var yamlToolsFS embed.FS

// loadYAMLTools walks the embedded package tree and parses every
// spwn.yaml it finds into an image.Tool instance. Directories
// without a manifest are silently skipped so non-package assets (e.g.
// README.md) don't break the load.
func loadYAMLTools() ([]ib.Tool, error) {
	entries, err := fs.ReadDir(yamlToolsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded catalog: %w", err)
	}
	var out []ib.Tool
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		manifestPath := path.Join(name, deps.Manifest)
		if _, err := fs.Stat(yamlToolsFS, manifestPath); err != nil {
			continue // directory without a manifest — legacy Go tool
		}
		names = append(names, name)
	}
	sort.Strings(names) // deterministic order
	for _, name := range names {
		parsed, err := deps.Parse(
			deps.EmbedResolver{FS: yamlToolsFS, Root: name},
			deps.ParseOptions{
				DefaultName:    "@spwn/" + strings.ReplaceAll(name, "_", "-"),
				DefaultVersion: "latest",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", name, err)
		}
		out = append(out, ib.ToolFromParsed(parsed))
	}
	return out, nil
}
