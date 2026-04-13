package node

import (
	"io/fs"

	ib "spwn.sh/packages/imagebuilder"
)

// Tool is the @spwn/node tool — Node.js 20 SDK.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/node" }
func (*tool) Kind() ib.Kind          { return ib.KindSDK }
func (*tool) Version() string        { return "20" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Commands: []string{
			"curl -fsSL https://deb.nodesource.com/setup_20.x | bash -",
			"apt-get install -y nodejs && rm -rf /var/lib/apt/lists/*",
		},
	}
}

func (*tool) Verify() []string {
	return []string{
		"command -v node",
		"command -v npm",
		"command -v npx",
	}
}

func (*tool) Skills() fs.FS { return nil }
