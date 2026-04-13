package build

import (
	"io/fs"

	ib "spwn.sh/packages/imagebuilder"
)

// Tool is the @spwn/build tool — C/C++ build essentials.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/build" }
func (*tool) Kind() ib.Kind          { return ib.KindSDK }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Packages: []string{"make", "gcc", "g++"},
	}
}

func (*tool) Verify() []string {
	return []string{
		"command -v make",
		"command -v gcc",
		"command -v g++",
	}
}

func (*tool) Skills() fs.FS { return nil }
