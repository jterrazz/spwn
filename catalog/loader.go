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
// dependencies (@spwn/unix, @spwn/git, …) and init-able project
// templates (@spwn/matrix, @spwn/startup, …). Every entry is
// registered as a Tool so `spwn install @spwn/<slug>` works
// uniformly; the `worlds:` section (when present) is consulted only
// by the init path and ignored by the image builder.
//
// Adding a new entry? Drop the directory in AND update this embed
// directive — Go's embed doesn't accept a bare wildcard because
// the package's own Go sources would otherwise land in the FS.
//
//go:embed all:architect all:build all:cli all:docker-cli all:git all:macrohard all:matrix all:mempalace all:node all:paperclip-factory all:python all:qmd all:research-lab all:startup all:unix
var catalogFS embed.FS

// loadYAMLTools walks every embedded entry that declares a spwn.yaml
// and parses it into an image.Tool. Every entry is registered — the
// image builder never reads the opaque `worlds:` field, so entries
// that double as project templates contribute an empty install spec
// but coexist cleanly with pure dependencies.
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
		manifestPath := path.Join(e.Name(), dependency.Manifest)
		if _, err := fs.Stat(catalogFS, manifestPath); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]ib.Tool, 0, len(names))
	for _, name := range names {
		canonical := "@spwn/" + name
		parsed, err := dependency.Parse(
			dependency.EmbedResolver{FS: catalogFS, Root: name},
			dependency.ParseOptions{
				DefaultName:    canonical,
				DefaultVersion: "latest",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", name, err)
		}
		// Catalog entries are always keyed by their canonical
		// @spwn/<slug> in the tool registry, regardless of what
		// spwn.yaml declares for `name:`. The file's name field is
		// free to be a user-facing project name (e.g. "matrix") that
		// survives `spwn init` verbatim.
		parsed.Schema.Name = canonical
		out = append(out, ib.ToolFromParsed(parsed))
	}
	return out, nil
}
