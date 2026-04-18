package spwn

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"spwn.sh/packages/dependency/internal/manifest"
	"spwn.sh/packages/dependency/tool"
)

// Every built-in catalog entry is mirrored here at `content/<slug>/`
// by the go:generate below — source-of-truth lives at /catalog/ in
// the repo root. The mirror is gitignored; `go generate` rehydrates
// it before every build. Go's go:embed directive can't walk `..`
// out of a package's directory, so the copy step is what keeps
// /catalog/ pure content while the loader lives here.
//
//go:generate bash -c "rm -rf content && mkdir content && cp -R ../../../../../catalog/. content/ && rm -f content/go.mod content/go.sum"
//go:embed all:content
var catalogFS embed.FS

// contentRoot is the directory prefix the go:generate mirror drops
// every source entry under. Paths flowing through the loader /
// gallery API stay relative to this root — consumers never see it.
const contentRoot = "content"

// loadYAMLTools walks every catalog entry that ships a
// tools/<slug>/tool.yaml and parses it into an tool.Tool. Project
// templates (entries whose root spwn.yaml declares `worlds:` but
// that don't ship a tool.yaml) are not tool-shaped and are not
// registered — they surface through the init gallery only.
func loadYAMLTools() ([]tool.Tool, error) {
	entries, err := fs.ReadDir(catalogFS, contentRoot)
	if err != nil {
		return nil, fmt.Errorf("read embedded catalog (is go:generate ran?): %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		toolPath := path.Join(contentRoot, e.Name(), "tools", e.Name(), manifest.ToolManifest)
		if _, err := fs.Stat(catalogFS, toolPath); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]tool.Tool, 0, len(names))
	for _, name := range names {
		canonical := "spwn:" + name
		parsed, err := manifest.Parse(
			manifest.EmbedResolver{FS: catalogFS, Root: path.Join(contentRoot, name, "tools", name)},
			manifest.ParseOptions{
				DefaultName:    canonical,
				DefaultVersion: "latest",
				ManifestFile:   manifest.ToolManifest,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", name, err)
		}
		// Catalog entries are always keyed by their canonical
		// spwn:<slug> in the tool registry, regardless of what
		// tool.yaml declares for `name:`.
		parsed.Schema.Name = canonical
		out = append(out, manifest.ToolFromParsed(parsed))
	}
	return out, nil
}
