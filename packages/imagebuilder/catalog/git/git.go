package git

import (
	"io/fs"

	ib "spwn.sh/packages/imagebuilder"
)

// Tool is the @spwn/git tool — Git version control.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/git" }
func (*tool) Kind() ib.Kind          { return ib.KindTool }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Packages: []string{"git"},
	}
}

func (*tool) Verify() []string {
	return []string{"command -v git"}
}

func (*tool) Skills() fs.FS { return nil }
