// Package mempalace is the @spwn/mempalace plugin: a local,
// raw-verbatim memory palace exposed through an MCP server.
//
// Integration: Claude Code reads ~/.claude/settings.json on startup
// and picks up the mcpServers entry injected by this plugin's
// Config("@spwn/claude-code"). The spawn-time merge pass in
// packages/world/internal/architect/spawn.go handles the injection;
// this package only declares what goes in.
package mempalace

import (
	"embed"
	"io/fs"

	ib "spwn.sh/packages/image"
)

//go:embed skills/*
var skills embed.FS

//go:embed config/claude-code.json
var claudeCodeConfig []byte

// Tool is the @spwn/mempalace plugin instance.
var Tool = &plugin{}

type plugin struct{}

func (*plugin) Name() string           { return "@spwn/mempalace" }
func (*plugin) Kind() ib.Kind          { return ib.KindTool }
func (*plugin) Version() string        { return "0.1.0" }
func (*plugin) Dependencies() []string { return []string{"@spwn/python"} }

func (*plugin) Install() ib.InstallSpec {
	// TODO: real-world usage may want a pinned version, a persistent
	// ChromaDB volume mount (mempalace init writes under ~/.mempalace),
	// and MEMPALACE_HOME set to a location inside /agents/<name>/ so
	// the palace survives container rebuilds. Left minimal for v1 —
	// the plugin boots, tests pass, and the live-QA pass can tune
	// paths once a real Claude key is used against it.
	return ib.InstallSpec{
		Commands: []string{
			"pip install --no-cache-dir --break-system-packages mempalace",
		},
	}
}

func (*plugin) Verify() []string {
	return []string{
		"command -v mempalace",
		"mempalace --version",
	}
}

func (*plugin) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}

// Runtimes declares which runtime backends this plugin targets. The
// spawn-time config-merge pass ignores plugins whose Runtimes() list
// doesn't include the world's backend.
func (*plugin) Runtimes() []string {
	return []string{"@spwn/claude-code"}
}

// Config returns the JSON snippet to merge into the named runtime's
// config file. For claude-code this is a `mcpServers` entry that
// points the runtime at the mempalace python MCP server.
func (*plugin) Config(runtime string) []byte {
	if runtime != "@spwn/claude-code" {
		return nil
	}
	return claudeCodeConfig
}
