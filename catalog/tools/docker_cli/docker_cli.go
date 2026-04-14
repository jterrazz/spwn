package docker_cli

import (
	"io/fs"

	ib "spwn.sh/packages/image"
)

// Tool is the @spwn/docker-cli tool - Docker CLI for DooD access.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string           { return "@spwn/docker-cli" }
func (*tool) Kind() ib.Kind          { return ib.KindTool }
func (*tool) Version() string        { return "latest" }
func (*tool) Dependencies() []string { return nil }

func (*tool) Install() ib.InstallSpec {
	return ib.InstallSpec{
		Packages: []string{"docker.io"},
	}
}

func (*tool) Verify() []string {
	return []string{"command -v docker"}
}

func (*tool) Skills() fs.FS { return nil }
