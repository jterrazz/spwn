package manifest

import (
	"io/fs"

	"spwn.sh/packages/dependency/tool"
)

// dependencyAdapter backs a parsed Schema as a tool.Tool.
type dependencyAdapter struct {
	schema    Schema
	fileBytes map[string][]byte
	skillsFS  fs.FS
}

// Name returns the fully-qualified ref (e.g. "spwn:git").
func (t *dependencyAdapter) Name() string { return t.schema.Name }

// Version returns the `version:` field, or the default the loader
// applied when the manifest left it blank.
func (t *dependencyAdapter) Version() string { return t.schema.Version }

// Dependencies returns the flat list of tool refs this one depends
// on. Resolution and topo sort happen in the registry.
func (t *dependencyAdapter) Dependencies() []string { return t.schema.Dependencies }

// Install converts the parsed InstallSection into the InstallSpec
// shape the image builder consumes. File bytes were read eagerly at
// parse time so this call is allocation-only.
func (t *dependencyAdapter) Install() tool.InstallSpec {
	spec := tool.InstallSpec{
		Packages: t.schema.Install.Packages,
		Commands: t.schema.Install.Commands,
		Env:      t.schema.Install.Env,
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
func (t *dependencyAdapter) Verify() []string { return t.schema.Verify }

// Skills returns an fs.FS rooted at the tool's skills/ directory,
// or nil when the directory is absent.
func (t *dependencyAdapter) Skills() fs.FS { return t.skillsFS }

// RuntimeProvider returns the name declared in `runtime-provider:`,
// or "" when none. Consumed by the spawn pipeline to look up a
// Go-registered provider for credential sync / default config files
// / prelaunch shell setup.
func (t *dependencyAdapter) RuntimeProvider() string {
	return t.schema.RuntimeProvider
}

// ToolFromParsed adapts a Parsed result into a tool.Tool.
// This is the bridge between the manifest domain and the rest of
// the codebase — adapters call this to hand Tool-shaped values to
// packages/compile's Registry.
func ToolFromParsed(p *Parsed) tool.Tool {
	skillsFS, _ := p.SkillsFS.(fs.FS)
	return &dependencyAdapter{
		schema:    p.Schema,
		fileBytes: p.FileBytes,
		skillsFS:  skillsFS,
	}
}
