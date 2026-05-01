package local

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"spwn.sh/packages/dependency/internal/manifest"
	"spwn.sh/packages/dependency/refs"
	"spwn.sh/packages/dependency/tool"
)

// localToolDir is where the project-local dependency loader looks
// for tool/<name> refs at image-build time. Mirrors what the
// validator (rulePacksExist) expects, so `spwn check` and
// `spwn build` resolve refs through the same on-disk layout.
const localToolDir = "tools"

// wrappedLocalTool forwards every tool.Tool method to an underlying
// packyaml-parsed dependency but forces Name() to the "local:<basename>"
// form. Catalog refs and local refs share a single registry keyed by
// name, so the `local:` prefix keeps them in separate namespaces —
// any future promotion of a local name to an spwn: dependency doesn't
// collide with existing `tool/` references in agent.yaml.
type wrappedLocalTool struct {
	inner tool.Tool
	name  string
}

func (t *wrappedLocalTool) Name() string              { return t.name }
func (t *wrappedLocalTool) Version() string           { return t.inner.Version() }
func (t *wrappedLocalTool) Install() tool.InstallSpec { return t.inner.Install() }
func (t *wrappedLocalTool) Verify() []string          { return t.inner.Verify() }
func (t *wrappedLocalTool) Skills() fs.FS             { return t.inner.Skills() }

// Dependencies rewrites `tool/<x>` inner deps to the `local:<x>`
// registry key so the resolver's lookups match the name under which
// we registered each local tool. Without this, a local tool whose
// tool.yaml lists `dependencies: [tool/b]` can't find tool/b —
// the registry only knows it as `local:b` — and the whole resolve
// step errors with "not registered", masking dependency cycles as
// missing-dep errors.
func (t *wrappedLocalTool) Dependencies() []string {
	raw := t.inner.Dependencies()
	out := make([]string, len(raw))
	for i, d := range raw {
		ref := refs.ParseRef(d)
		if ref.Kind == refs.KindLocalTool {
			out[i] = "local:" + ref.Name
		} else {
			out[i] = d
		}
	}
	return out
}

// LoadTool parses spwn/tools/<name>/tool.yaml via the
// shared packyaml parser and wraps the result so Name() returns
// "local:<name>". Missing manifest is a crisp authoring error — an
// empty local dependency would render to nothing and the user would
// spend an afternoon debugging a no-op.
func LoadTool(projectRoot, name string) (tool.Tool, error) {
	pkgDir := filepath.Join(projectRoot, "spwn", localToolDir, name)
	info, err := os.Stat(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("local dependency %q: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local dependency %q: %s is not a directory", name, pkgDir)
	}

	parsed, err := manifest.Parse(
		manifest.DirResolver{Root: pkgDir},
		manifest.ParseOptions{
			DefaultName:    name,
			DefaultVersion: "0.0.0-local",
			ManifestFile:   manifest.ToolManifest,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("local dependency %q: %w", name, err)
	}

	return &wrappedLocalTool{inner: manifest.ToolFromParsed(parsed), name: "local:" + name}, nil
}

// Hydrate walks a flat list of dependency refs, loads
// every tool/<name> entry as a synthetic tool.Tool via the shared
// packyaml parser, registers it, and returns the rewritten list
// where each tool/ ref has been replaced by its "local:<name>"
// registry key.
//
// skill/, hook/, and command/ refs are stripped from the list
// entirely — those are compile-time artifacts that the runtime
// renderer weaves into the Tree (as /world/skills/<name>/SKILL.md,
// hook entries in settings.json/hooks.json, and slash-invoked
// commands respectively), not image-builder inputs. Passing them
// through to the image registry's Resolve would blow up with "tool
// not found" because they're never registered there.
//
// spwn: and github: refs pass through unchanged for the image
// resolver to handle.
//
// Order is preserved so users see their ref list echoed back in the
// same shape they declared it. Duplicates are tolerated — the
// registry's Register is called once per unique name.
func Hydrate(reg tool.Registry, projectRoot string, depRefs []string) ([]string, error) {
	loaded := map[string]bool{}

	// hydrateOne loads one tool/<name> (recursively following its
	// own `tool/` deps so the registry ends up with every local
	// tool reachable from the project). Deduplicated via `loaded`;
	// returns the registry key ("local:<name>") for the caller.
	var hydrateOne func(name string) (string, error)
	hydrateOne = func(name string) (string, error) {
		if loaded[name] {
			return "local:" + name, nil
		}
		t, err := LoadTool(projectRoot, name)
		if err != nil {
			return "", err
		}
		loaded[name] = true

		// Collect inner deps BEFORE wrapping swaps "tool/" →
		// "local:" on the returned slice. Type-assert down to the
		// wrapper we constructed in LoadTool so we can read the
		// original Dependencies() entries that still carry the
		// `tool/<x>` ref form.
		var innerDeps []string
		if wrapped, ok := t.(*wrappedLocalTool); ok {
			innerDeps = wrapped.inner.Dependencies()
		} else {
			innerDeps = t.Dependencies()
		}

		if err := reg.Register(t); err != nil {
			return "", fmt.Errorf("register local dependency %q: %w", name, err)
		}

		// Recurse into local-tool inner deps so the registry has
		// every transitively-reachable local tool. Without this, a
		// local tool whose tool.yaml declares `dependencies: [tool/b]`
		// causes resolver.Resolve to error with "local:b not
		// registered" at spawn, masking real cycles behind a
		// missing-dep message.
		for _, innerRaw := range innerDeps {
			innerRef := refs.ParseRef(innerRaw)
			if innerRef.Kind == refs.KindLocalTool && innerRef.Name != "" {
				if _, ierr := hydrateOne(innerRef.Name); ierr != nil {
					return "", ierr
				}
			}
		}
		return "local:" + name, nil
	}

	out := make([]string, 0, len(depRefs))
	for _, raw := range depRefs {
		ref := refs.ParseRef(raw)
		switch ref.Kind {
		case refs.KindLocalSkill, refs.KindLocalHook, refs.KindLocalCommand:
			// Strip — compile step consumes these, image builder
			// doesn't know how to.
			continue
		case refs.KindLocalTool:
			// Fall through to the hydrate path below.
		default:
			// spwn:, github:, KindInvalid — pass through untouched.
			out = append(out, raw)
			continue
		}

		if ref.Name == "" {
			// Malformed local ref — let Resolve surface a clear
			// "unknown tool" error rather than crashing on an empty
			// filesystem lookup.
			out = append(out, raw)
			continue
		}
		key, err := hydrateOne(ref.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, nil
}
