// This file is the package's public operation surface — one door
// for every dependency-resolution concern. Adapters (spwn catalog,
// local blocks, future registries) live hidden under internal/ and
// never appear in caller import paths.
//
// Every exported function here either:
//   - enumerates tools from one or more adapters (Tools, Gallery),
//   - installs or loads by ref from the right adapter (Install,
//     LoadLocalTool),
//   - or binds adapter output into a caller-supplied target
//     (RegisterBuiltins, HydrateLocals).
//
// Callers never need to know which adapter produced a given Tool —
// the Tool interface is the contract.

package dependency

import (
	"spwn.sh/packages/dependency/internal/adapters/local"
	"spwn.sh/packages/dependency/internal/adapters/spwn"
	"spwn.sh/packages/dependency/tool"
)

// GalleryEntry is the public-facing description of one catalog
// entry that ships a `worlds:` section — installable as a project
// template via `spwn init <slug>`. Adapter-backed type aliased
// through so external callers stay on one import.
type GalleryEntry = spwn.Example

// InstallReport describes everything Install wrote to disk.
type InstallReport = spwn.InstallReport

// ErrNotFound is returned by gallery lookups when the slug doesn't
// match any entry. Callers compare with errors.Is (or == since the
// underlying is a plain sentinel).
var ErrNotFound = spwn.ErrNotFound

// ── Builtin catalog (the spwn adapter) ─────────────────────────────

// BuiltinTools returns every builtin tool shipped in the compiled binary
// (the spwn: catalog). These are ref-by-name installables —
// spwn:unix, spwn:python, etc. Project-template entries (matrix,
// severance, …) are excluded; see Gallery for those.
func BuiltinTools() []tool.Tool { return spwn.All }

// RegisterBuiltins registers every builtin tool into the given
// target registry. Convenience wrapper around iterating Tools()
// and calling target.Register for each. Returns the first
// registration error (typically a duplicate name — a programmer
// error in the catalog).
func RegisterBuiltins(target tool.Registry) error {
	for _, t := range spwn.All {
		if err := target.Register(t); err != nil {
			return err
		}
	}
	return nil
}

// ── Gallery (init-able catalog entries) ────────────────────────────

// Gallery returns every init-able catalog entry with metadata,
// sorted in canonical display order (startup first, then matrix,
// then the rest alphabetically).
func Gallery() ([]GalleryEntry, error) { return spwn.List() }

// GalleryEntryBySlug returns one gallery entry's metadata or
// ErrNotFound when the slug is not a gallery entry (either it
// doesn't exist or it's a dep-only entry without `worlds:`).
func GalleryEntryBySlug(slug string) (GalleryEntry, error) { return spwn.Get(slug) }

// GallerySlugs returns just the slug names of gallery entries —
// the bare-name lookup set for `spwn init <slug>` resolution.
// Sorted in display order.
func GallerySlugs() []string { return spwn.ShippedSlugs() }

// Install materialises a gallery entry into baseDir as a new
// project tree. Existing files are never overwritten (re-install
// is idempotent). Returns an InstallReport listing what was added
// vs skipped.
func Install(slug, baseDir string) (InstallReport, error) { return spwn.Install(slug, baseDir) }

// InstallInto installs into the active project root when one is
// discoverable, else into the user-global ~/.spwn (legacy global
// mode). Convenience wrapper for the CLI.
func InstallInto(slug string) (InstallReport, error) { return spwn.InstallInto(slug) }

// CopyGateTools materialises every catalog tool under spwn:<refName>
// that has a `gate:` section into `<gateToolsRoot>/<short>/`. Called
// from the install CLI immediately after the manifest mutation so a
// `spwn install spwn:x` followed by `spwn gate restart` picks up
// the new tool with no manual file dance. Idempotent — overwrites
// existing files so catalog updates flow through on re-install.
//
// Returns the list of slugs copied (empty when the entry has no
// gate-shaped tools — most catalog entries are agent-side only).
func CopyGateTools(refName, gateToolsRoot string) ([]string, error) {
	return spwn.CopyGateTools(refName, gateToolsRoot)
}

// ── Project-local blocks (the local adapter) ──────────────────────

// LoadLocalTool reads spwn/tools/<name>/ inside projectRoot and
// returns a Tool. Used by `spwn inspect` to render local
// compositions and by the spawn pipeline to register locals.
// Returns an error with a clear hint when tool.yaml is missing.
func LoadLocalTool(projectRoot, name string) (tool.Tool, error) {
	return local.LoadTool(projectRoot, name)
}

// HydrateLocals filters a dep ref list for `tool:<name>` entries,
// loads each from <projectRoot>/spwn/tools/<name>/, registers it
// into target, and returns the rewritten list where each `tool:`
// ref is replaced by its registered `local:<name>` key. `skill:`
// and `hook:` refs are stripped (consumed elsewhere — the image
// builder has no use for them). `spwn:` and `github:` refs pass
// through untouched.
//
// Order is preserved so callers see their ref list echoed back in
// the same shape they declared it.
func HydrateLocals(target tool.Registry, projectRoot string, refs []string) ([]string, error) {
	return local.Hydrate(target, projectRoot, refs)
}
