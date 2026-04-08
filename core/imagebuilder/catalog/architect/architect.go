package architect

import (
	"embed"
	"io/fs"

	ib "spwn.sh/core/imagebuilder"
)

//go:embed skills/*
var skills embed.FS

//go:embed files/*
var files embed.FS

// Tool is the @spwn/architect tool — the always-on orchestration daemon.
var Tool = &tool{}

type tool struct{}

func (*tool) Name() string    { return "@spwn/architect" }
func (*tool) Kind() ib.Kind   { return ib.KindPlatform }
func (*tool) Version() string { return "1.1.0" }
func (*tool) Dependencies() []string {
	return []string{"@spwn/cli", "@spwn/claude-code", "@spwn/docker-cli"}
}

func (*tool) Install() ib.InstallSpec {
	entrypoint, _ := fs.ReadFile(files, "files/entrypoint.sh")

	return ib.InstallSpec{
		Files: map[string][]byte{
			"/usr/local/bin/architect-entrypoint.sh": entrypoint,
		},
		Commands: []string{
			"chmod +x /usr/local/bin/architect-entrypoint.sh",
		},
	}
}

func (*tool) Verify() []string {
	return []string{
		"command -v spwn",
		"command -v claude",
		"command -v docker",
	}
}

func (*tool) Skills() fs.FS {
	sub, _ := fs.Sub(skills, "skills")
	return sub
}
