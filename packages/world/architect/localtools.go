package architect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/image/packyaml"
)

// localPackDir is where the project-local pack loader looks
// for bare-name refs at image-build time. Mirrors what the validator
// (rulePacksExist) expects, so `spwn check` and `spwn build`
// resolve refs through the same on-disk layout.
const localPackDir = "packs"

// wrappedLocalTool forwards every image.Tool method to an underlying
// packyaml-parsed pack but forces Name() to the "local:<basename>"
// form. Catalog refs and local refs share a single registry keyed by
// name, so the prefix keeps them in separate namespaces — any future
// promotion of a local name to an @spwn/ pack doesn't collide with
// existing bare-name references in agent.yaml.
type wrappedLocalTool struct {
	inner ib.Tool
	name  string
}

func (t *wrappedLocalTool) Name() string                { return t.name }
func (t *wrappedLocalTool) Kind() ib.Kind               { return ib.KindTool }
func (t *wrappedLocalTool) Version() string             { return t.inner.Version() }
func (t *wrappedLocalTool) Dependencies() []string      { return t.inner.Dependencies() }
func (t *wrappedLocalTool) Install() ib.InstallSpec     { return t.inner.Install() }
func (t *wrappedLocalTool) Verify() []string            { return t.inner.Verify() }
func (t *wrappedLocalTool) Skills() fs.FS               { return t.inner.Skills() }
func (t *wrappedLocalTool) Runtimes() []string          { return t.inner.Runtimes() }
func (t *wrappedLocalTool) Config(runtime string) []byte { return t.inner.Config(runtime) }

// loadLocalPack parses spwn/packs/<name>/pack.yaml via the
// shared pluginyaml parser and wraps the result so Name() returns
// "local:<name>". Missing manifest is a crisp authoring error — an
// empty local pack would render to nothing and the user would
// spend an afternoon debugging a no-op.
func loadLocalPack(projectRoot, name string) (ib.Tool, error) {
	pkgDir := filepath.Join(projectRoot, "spwn", localPackDir, name)
	info, err := os.Stat(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("local pack %q: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local pack %q: %s is not a directory", name, pkgDir)
	}

	tool, err := packyaml.Parse(
		packyaml.DirResolver{Root: pkgDir},
		packyaml.ParseOptions{
			DefaultName:    name,
			DefaultVersion: "0.0.0-local",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("local pack %q: %w", name, err)
	}

	return &wrappedLocalTool{inner: tool, name: "local:" + name}, nil
}

// hydrateLocalPacks walks a flat list of pack refs, loads
// every bare (non-@) name as a synthetic image.Tool via the shared
// pluginyaml parser, registers it, and returns the rewritten list
// where each bare name has been replaced by its "local:<name>"
// registry key.
//
// Order is preserved so users see their ref list echoed back in the
// same shape they declared it. Duplicates are tolerated — the
// registry's Register is called once per unique name.
func hydrateLocalPacks(reg *ib.Registry, projectRoot string, refs []string) ([]string, error) {
	out := make([]string, 0, len(refs))
	loaded := map[string]bool{}
	for _, raw := range refs {
		if strings.HasPrefix(raw, "@") {
			out = append(out, raw)
			continue
		}
		if loaded[raw] {
			out = append(out, "local:"+raw)
			continue
		}
		tool, err := loadLocalPack(projectRoot, raw)
		if err != nil {
			return nil, err
		}
		if err := reg.Register(tool); err != nil {
			return nil, fmt.Errorf("register local pack %q: %w", raw, err)
		}
		loaded[raw] = true
		out = append(out, "local:"+raw)
	}
	return out, nil
}
