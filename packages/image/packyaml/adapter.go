package packyaml

import (
	"io/fs"

	ib "spwn.sh/packages/image"
)

// toolImpl backs a parsed Schema as an image.Tool. Runtimes() and
// Config() are part of the unified Tool interface — a pack with a
// `runtime-config:` block returns a non-empty Runtimes list and the spawn-time
// merger picks up its Config(runtime) snippet. Packs without a
// runtime-config block return nil from both.
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
		AptPackages:  t.schema.Install.AptPackages,
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

// Runtimes returns the runtime backends this pack targets for
// runtime-config injection. Returns nil when the manifest has no
// `runtime-config:` block, which the spawn-time merger reads as "not a pack."
func (t *toolImpl) Runtimes() []string {
	if t.schema.RuntimeConfig == nil {
		return nil
	}
	return t.schema.RuntimeConfig.Runtimes
}

// Config returns the JSON bytes for the requested runtime's config
// snippet, or nil when this pack has no runtime-config block or no config
// for that runtime.
func (t *toolImpl) Config(runtime string) []byte {
	if t.schema.RuntimeConfig == nil {
		return nil
	}
	// Enforce Runtimes() allowlist at the boundary so individual
	// tools don't have to.
	match := false
	for _, r := range t.schema.RuntimeConfig.Runtimes {
		if r == runtime {
			match = true
			break
		}
	}
	if !match {
		return nil
	}
	b, err := t.schema.RuntimeConfig.ConfigJSON(runtime)
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
