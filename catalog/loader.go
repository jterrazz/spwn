package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"spwn.sh/packages/dependency"
	ib "spwn.sh/packages/image"
)

// catalogFS embeds every built-in catalog entry — both installable
// dependencies (@spwn/unix, @spwn/git, …) and init-able project
// templates (@spwn/matrix, @spwn/startup, …). Entries are classified
// at load time by the presence of example.yaml; the verb the user
// invokes (`spwn install` vs `spwn init`) picks which face of the
// entry is used.
//
// Adding a new entry? Drop the directory in AND update this embed
// directive — Go's embed doesn't accept a bare wildcard because
// the package's own Go sources would otherwise land in the FS.
//
//go:embed all:build all:docker_cli all:git all:macrohard all:matrix all:mempalace all:node all:paperclip-factory all:python all:qmd all:research-lab all:spwn_architect all:spwn_cli all:startup all:unix
var catalogFS embed.FS

// loadYAMLTools walks every embedded entry that declares a spwn.yaml
// and parses it into an image.Tool. Entries that ship an example.yaml
// sidecar are project templates, not installable dependencies — they
// are surfaced through List/Get/Install (for `spwn init <slug>`) and
// deliberately excluded here so `spwn install @spwn/<slug>` only
// accepts dep-shaped entries.
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
			continue // directory without a spwn.yaml — not an entry
		}
		if _, err := fs.Stat(catalogFS, path.Join(e.Name(), "example.yaml")); err == nil {
			continue // example/template entry, not an installable dep
		}
		names = append(names, e.Name())
	}
	sort.Strings(names) // deterministic order

	out := make([]ib.Tool, 0, len(names))
	for _, name := range names {
		parsed, err := dependency.Parse(
			dependency.EmbedResolver{FS: catalogFS, Root: name},
			dependency.ParseOptions{
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
