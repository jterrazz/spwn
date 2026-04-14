package spwn_cli

import (
	"embed"
	"io/fs"

	ib "spwn.sh/packages/image"
)

//go:embed skills/*
var skills embed.FS

// Tool is the @spwn/cli tool - spwn CLI for managing worlds and agents.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/cli" }
func (*tool) Kind() ib.Kind          { return ib.KindPlatform }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{
			"curl -fsSL https://spwn.sh/install.sh | bash || true",
		},
	}
}

func (*tool) Verify() []string {
	return []string{"command -v spwn"}
}

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}
