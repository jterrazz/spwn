package toolyaml

import (
	"io/fs"

	ib "spwn.sh/packages/image"
)

// toolImpl backs a parsed Schema as an image.Tool. When the manifest
// includes a `plugin:` block, toolImpl also satisfies image.Plugin so
// type assertions like `if p, ok := t.(ib.Plugin); ok { ... }` work
// without special-casing YAML-backed tools.
type toolImpl struct {
	schema    Schema
	kind      ib.Kind
	fileBytes map[string][]byte
	skillsFS  fs.FS
}

// Name returns the fully-qualified ref (e.g. "@spwn/git").
func (t *toolImpl) Name() string { return t.schema.Name }

// Kind returns the classification parsed from the `kind:` field.
func (t *toolImpl) Kind() ib.Kind { return t.kind }

// Version returns the `version:` field, or the default the loader
// applied when the manifest left it blank.
func (t *toolImpl) Version() string { return t.schema.Version }

// Dependencies returns the flat list of tool refs this one depends
// on. Resolution and topo sort happen in the registry.
func (t *toolImpl) Dependencies() []string { return t.schema.Dependencies }

// Install converts the parsed InstallSection into the InstallSpec
// shape the image builder consumes. File bytes were read eagerly at
// parse time so this call is allocation-only.
func (t *toolImpl) Install() ib.InstallSpec {
	spec := ib.InstallSpec{
		Packages:     t.schema.Install.Packages,
		Commands:     t.schema.Install.Commands,
		UserCommands: t.schema.Install.UserCommands,
		Env:          t.schema.Install.Env,
	}
	if len(t.fileBytes) > 0 {
		spec.Files = make(map[string][]byte, len(t.fileBytes))
		for k, v := range t.fileBytes {
			spec.Files[k] = v
		}
	}
	return spec
}

// Verify returns the post-build sanity commands from `verify:`.
func (t *toolImpl) Verify() []string { return t.schema.Verify }

// Skills returns an fs.FS rooted at the tool's skills/ directory,
// or nil when the directory is absent.
func (t *toolImpl) Skills() fs.FS { return t.skillsFS }

// Runtimes satisfies the image.Plugin interface. Returns nil when the
// manifest has no `plugin:` block, which collapses the type assertion
// `_, ok := t.(ib.Plugin)` to "not a plugin" in the caller's eye.
//
// Note: because every *toolImpl has this method, every toolImpl
// technically satisfies image.Plugin at the type system level. Call
// sites that want to distinguish "plugin" from "plain tool" should
// check `len(Runtimes()) > 0` rather than relying on a type assertion.
// image.PluginRuntimes already does this check internally so most
// call sites are fine.
func (t *toolImpl) Runtimes() []string {
	if t.schema.Plugin == nil {
		return nil
	}
	return t.schema.Plugin.Runtimes
}

// Config satisfies image.Plugin. Returns the JSON bytes for the
// requested runtime's config snippet, or nil when this tool is not a
// plugin or has no config for that runtime.
func (t *toolImpl) Config(runtime string) []byte {
	if t.schema.Plugin == nil {
		return nil
	}
	// Enforce Runtimes() allowlist at the boundary so individual
	// tools don't have to.
	match := false
	for _, r := range t.schema.Plugin.Runtimes {
		if r == runtime {
			match = true
			break
		}
	}
	if !match {
		return nil
	}
	b, err := t.schema.Plugin.ConfigJSON(runtime)
	if err != nil {
		return nil
	}
	return b
}

// RuntimeProvider returns the name declared in `runtime-provider:`,
// or "" when none. Consumed by the spawn pipeline to look up a
// Go-registered provider for credential sync / default config files
// / prelaunch shell setup.
func (t *toolImpl) RuntimeProvider() string {
	return t.schema.RuntimeProvider
}
