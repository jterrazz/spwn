package qmd

import (
	"embed"
	"io/fs"

	ib "spwn.sh/core/imagebuilder"
)

//go:embed skills/*
var skills embed.FS

// Tool is the @qmd tool — on-device markdown search engine.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@qmd" }
func (*tool) Kind() ib.Kind          { return ib.KindTool }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return []string{"@node"} }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{"npm install -g @tobilu/qmd"},
	}
}

func (*tool) Verify() []string {
	return []string{"command -v qmd"}
}

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}
