package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"spwn.sh/packages/dependency"
	ib "spwn.sh/packages/image"
)

// catalogFS embeds every built-in catalog entry — both installable
// dependencies (spwn:unix, spwn:git, …) and init-able project
// templates (spwn:matrix, spwn:startup, …). Every entry ships a
// root spwn.yaml (project-shape); pure-dep entries additionally
// carry the actual tool definition at tools/<slug>/tool.yaml.
//
// Adding a new entry? Drop the directory in AND update this embed
// directive — Go's embed doesn't accept a bare wildcard because
// the package's own Go sources would otherwise land in the FS.
//
//go:embed all:architect all:build all:cli all:docker-cli all:git all:macrohard all:matrix all:mempalace all:node all:paperclip-factory all:python all:qmd all:research-lab all:startup all:unix
var catalogFS embed.FS

// loadYAMLTools walks every catalog entry that ships a
// tools/<slug>/tool.yaml and parses it into an image.Tool. Project
// templates (entries whose root spwn.yaml declares `worlds:` but
// that don't ship a tool.yaml) are not tool-shaped and are not
// registered — they surface through the init gallery only.
func loadYAMLTools() ([]ib.Tool, error) {
	entries, err := fs.ReadDir(catalogFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded catalog: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// A pure-dep entry is <slug>/tools/<slug>/tool.yaml.
		toolPath := path.Join(e.Name(), "tools", e.Name(), dependency.ToolManifest)
		if _, err := fs.Stat(catalogFS, toolPath); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]ib.Tool, 0, len(names))
	for _, name := range names {
		canonical := "spwn:" + name
		parsed, err := dependency.Parse(
			dependency.EmbedResolver{FS: catalogFS, Root: path.Join(name, "tools", name)},
			dependency.ParseOptions{
				DefaultName:    canonical,
				DefaultVersion: "latest",
				ManifestFile:   dependency.ToolManifest,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", name, err)
		}
		// Catalog entries are always keyed by their canonical
		// spwn:<slug> in the tool registry, regardless of what
		// tool.yaml declares for `name:`.
		parsed.Schema.Name = canonical
		out = append(out, ib.ToolFromParsed(parsed))
	}
	return out, nil
}
