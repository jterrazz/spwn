package architect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/dependency"
)

// localToolDir is where the project-local dependency loader looks
// for bare-name refs at image-build time. Mirrors what the validator
// (rulePacksExist) expects, so `spwn check` and `spwn build`
// resolve refs through the same on-disk layout.
const localToolDir = "tools"

// wrappedLocalTool forwards every image.Tool method to an underlying
// packyaml-parsed dependency but forces Name() to the "local:<basename>"
// form. Catalog refs and local refs share a single registry keyed by
// name, so the prefix keeps them in separate namespaces — any future
// promotion of a local name to an @spwn/ dependency doesn't collide with
// existing bare-name references in agent.yaml.
type wrappedLocalTool struct {
	inner ib.Tool
	name  string
}

func (t *wrappedLocalTool) Name() string                { return t.name }
func (t *wrappedLocalTool) Kind() dependency.Kind               { return dependency.KindTool }
func (t *wrappedLocalTool) Version() string             { return t.inner.Version() }
func (t *wrappedLocalTool) Dependencies() []string      { return t.inner.Dependencies() }
func (t *wrappedLocalTool) Install() ib.InstallSpec     { return t.inner.Install() }
func (t *wrappedLocalTool) Verify() []string            { return t.inner.Verify() }
func (t *wrappedLocalTool) Skills() fs.FS               { return t.inner.Skills() }
func (t *wrappedLocalTool) Runtimes() []string          { return t.inner.Runtimes() }
func (t *wrappedLocalTool) Config(runtime string) []byte { return t.inner.Config(runtime) }

// loadLocalPack parses spwn/tools/<name>/tool.yaml via the
// shared packyaml parser and wraps the result so Name() returns
// "local:<name>". Missing manifest is a crisp authoring error — an
// empty local dependency would render to nothing and the user would
// spend an afternoon debugging a no-op.
func loadLocalPack(projectRoot, name string) (ib.Tool, error) {
	pkgDir := filepath.Join(projectRoot, "spwn", localToolDir, name)
	info, err := os.Stat(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("local dependency %q: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local dependency %q: %s is not a directory", name, pkgDir)
	}

	parsed, err := dependency.Parse(
		dependency.DirResolver{Root: pkgDir},
		dependency.ParseOptions{
			DefaultName:    name,
			DefaultVersion: "0.0.0-local",
			ManifestFile:   dependency.ToolManifest,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("local dependency %q: %w", name, err)
	}

	return &wrappedLocalTool{inner: ib.ToolFromParsed(parsed), name: "local:" + name}, nil
}

// hydrateLocalPacks walks a flat list of dependency refs, loads
// every bare (non-@) name as a synthetic image.Tool via the shared
// packyaml parser, registers it, and returns the rewritten list
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
		// ParseRef classifies every syntax: @spwn/x, spwn:x,
		// github:a/b, @a/b → non-local; bare name → local.
		// Anything non-local passes through for the registry to
		// resolve via its catalog entry.
		ref := dependency.ParseRef(raw)
		if ref.Kind != dependency.KindLocal {
			out = append(out, raw)
			continue
		}
		name := ref.Name
		if name == "" {
			// Malformed local ref — let Resolve surface a clear
			// "unknown tool" error rather than crashing on an empty
			// filesystem lookup.
			out = append(out, raw)
			continue
		}
		if loaded[name] {
			out = append(out, "local:"+name)
			continue
		}
		tool, err := loadLocalPack(projectRoot, name)
		if err != nil {
			return nil, err
		}
		if err := reg.Register(tool); err != nil {
			return nil, fmt.Errorf("register local dependency %q: %w", name, err)
		}
		loaded[name] = true
		out = append(out, "local:"+name)
	}
	return out, nil
}
