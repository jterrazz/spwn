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

// outside: go:embed cannot walk `..` out of the package directory, so go:generate mirrors /catalog/ into content/
//
//go:generate bash -c "rm -rf content && mkdir content && cp -R ../../../../../catalog/. content/ && rm -f content/go.mod content/go.sum"
//go:embed all:content
var catalogFS embed.FS

const contentRoot = "content"

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
		parsed.Schema.Name = canonical
		out = append(out, manifest.ToolFromParsed(parsed))
	}
	return out, nil
}
