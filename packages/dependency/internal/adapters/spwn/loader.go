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

// loadYAMLTools walks every catalog entry that ships one or more
// tools/<name>/tool.yaml and parses each one into a tool.Tool.
//
// Per-tool layout: each tool's resolver is rooted at its own
// tools/<name>/ dir — tool.yaml and anything it references via
// `files:` resolve locally. The SkillsRoot override points skills
// back at the catalog-entry-level skills/ directory so skills stay
// exposed alongside tools/ and are shared across every tool in the
// bundle.
//
// Multi-tool ready: the loader walks every tools/<subdir>/, so a
// catalog entry shipping two tools (tools/a/, tools/b/) registers
// both without further code changes.
//
// Project templates (entries whose root spwn.yaml declares `worlds:`
// but that don't ship a tools/<name>/tool.yaml) are not tool-shaped
// and are not registered — they surface through the init gallery
// only.
func loadYAMLTools() ([]tool.Tool, error) {
	entries, err := fs.ReadDir(catalogFS, contentRoot)
	if err != nil {
		return nil, fmt.Errorf("read embedded catalog (did you run `make generate` or `go generate ./packages/dependency/...`?): %w", err)
	}

	type toolLoc struct {
		entry, tool string
	}
	var locs []toolLoc
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Template entries (spwn.yaml declares worlds:) ship tools
		// that are scoped to the template, not global deps. Their
		// tools/<name>/ dirs become spwn/tools/<name>/ in the user
		// project on install; they don't register as spwn:<tool>
		// refs.
		if schema, err := loadEntrySchema(e.Name()); err == nil && hasWorlds(schema) {
			continue
		}
		toolsDir := path.Join(contentRoot, e.Name(), "tools")
		subs, err := fs.ReadDir(catalogFS, toolsDir)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			if !sub.IsDir() {
				continue
			}
			toolPath := path.Join(toolsDir, sub.Name(), manifest.ToolManifest)
			if _, err := fs.Stat(catalogFS, toolPath); err != nil {
				continue
			}
			locs = append(locs, toolLoc{entry: e.Name(), tool: sub.Name()})
		}
	}
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].entry != locs[j].entry {
			return locs[i].entry < locs[j].entry
		}
		return locs[i].tool < locs[j].tool
	})

	out := make([]tool.Tool, 0, len(locs))
	for _, loc := range locs {
		canonical := "spwn:" + loc.tool
		parsed, err := manifest.Parse(
			manifest.EmbedResolver{
				FS:         catalogFS,
				Root:       path.Join(contentRoot, loc.entry, "tools", loc.tool),
				SkillsRoot: path.Join(contentRoot, loc.entry, "skills"),
			},
			manifest.ParseOptions{
				DefaultName:    canonical,
				DefaultVersion: "latest",
				ManifestFile:   manifest.ToolManifest,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load %s/%s: %w", loc.entry, loc.tool, err)
		}
		// Catalog entries key tools by their canonical spwn:<tool>
		// regardless of what tool.yaml declares for `name:`.
		parsed.Schema.Name = canonical
		out = append(out, manifest.ToolFromParsed(parsed))
	}
	return out, nil
}
