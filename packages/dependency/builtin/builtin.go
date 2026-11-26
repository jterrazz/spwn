package builtin

import (
	"spwn.sh/packages/dependency/internal/adapters/local"
	"spwn.sh/packages/dependency/internal/adapters/spwn"
	"spwn.sh/packages/dependency/tool"
)

// BuiltinTools returns every builtin tool shipped in the compiled binary
// (the spwn: catalog). These are ref-by-name installables:
// spwn:unix, spwn:python, etc.
func BuiltinTools() []tool.Tool { return spwn.All }

// RegisterBuiltins registers every builtin tool into the given
// target registry. Convenience wrapper around iterating Tools()
// and calling target.Register for each. Returns the first
// registration error (typically a duplicate name, a programmer
// error in the catalog).
func RegisterBuiltins(target tool.Registry) error {
	for _, t := range spwn.All {
		if err := target.Register(t); err != nil {
			return err
		}
	}
	return nil
}

// CopyGateTools materialises every catalog tool under spwn:<refName>
// that has a `gate:` section into `<gateToolsRoot>/<short>/`. Called
// from the install CLI immediately after the manifest mutation so a
// `spwn install spwn:x` followed by `spwn gate restart` picks up
// the new tool with no manual file dance. Idempotent: it overwrites
// existing files so catalog updates flow through on re-install.
//
// Returns the list of slugs copied (empty when the entry has no
// gate-shaped tools; most catalog entries are agent-side only).
func CopyGateTools(refName, gateToolsRoot string) ([]string, error) {
	return spwn.CopyGateTools(refName, gateToolsRoot)
}

// HydrateLocals filters a dep ref list for `tool/<name>` entries,
// loads each from <projectRoot>/spwn/tools/<name>/, registers it
// into target, and returns the rewritten list where each `tool/`
// ref is replaced by its registered `local:<name>` key. `skill/`
// and `hook/` refs are stripped (consumed elsewhere: the image
// builder has no use for them). `spwn:` and `github:` refs pass
// through untouched.
//
// Order is preserved so callers see their ref list echoed back in
// the same shape they declared it.
func HydrateLocals(target tool.Registry, projectRoot string, refs []string) ([]string, error) {
	return local.Hydrate(target, projectRoot, refs)
}
