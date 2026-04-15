package python

import (
	"io/fs"

	ib "spwn.sh/packages/image"
)

// Tool is the @spwn/python tool - Python 3 SDK.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/python" }
func (*tool) Kind() ib.Kind          { return ib.KindSDK }
func (*tool) Version() string        { return "3" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Packages: []string{"python3", "python3-pip"},
	}
}

func (*tool) Verify() []string {
	return []string{
		"command -v python3",
		"command -v pip3",
	}
}

func (*tool) Skills() fs.FS { return nil }
