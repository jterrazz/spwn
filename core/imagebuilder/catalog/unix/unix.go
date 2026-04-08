package unix

import (
	"io/fs"

	ib "spwn.sh/core/imagebuilder"
)

// Tool is the @unix tool — core Unix utilities.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string    { return "@unix" }
func (*tool) Kind() ib.Kind   { return ib.KindSDK }
func (*tool) Version() string { return "24.04" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Packages: []string{
			"bash", "coreutils", "findutils", "grep", "sed", "gawk",
			"curl", "wget", "jq",
		},
	}
}

func (*tool) Verify() []string {
	return []string{
		"command -v bash",
		"command -v grep",
		"command -v sed",
		"command -v awk",
		"command -v curl",
		"command -v jq",
	}
}

func (*tool) Skills() fs.FS { return nil }
