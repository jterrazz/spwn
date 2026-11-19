package zshblock

import (
	"context"
	"io/fs"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type BlockOptions struct {
	ThisMachine catalog.MachineKind
	PortsEnv    string
}

type Block struct {
	// value: the script station's ~/.zshrc block evaluates: the dotfiles'
	// loader, the tools' activation, completion, the granted programs'
	// environment files and the services' variables.
	files    Files
	embedded fs.FS
	o        BlockOptions
}

func NewBlock(files Files, embedded fs.FS, o BlockOptions) *Block {
	return &Block{files: files, embedded: embedded, o: o}
}

func (s *Block) Zsh(_ context.Context, completion string) (string, error) {
	loader, err := fs.ReadFile(s.embedded, catalog.ZshLoader)
	if err != nil {
		return "", err
	}
	// invariant: an interactive shell that ran no compinit (a fresh Mac, no oh-my-zsh)
	// gets it from the block, so the completion is there; a script's shell runs none,
	// and the completion is guarded by `$+functions[compdef]`.
	script := string(loader) + "\n" + catalog.AppActivation + "\n" +
		"if (( ! $+functions[compdef] )) && [[ -o interactive ]]; then\n  autoload -Uz compinit && compinit -i\nfi\n" +
		"if (( $+functions[compdef] )); then\n" + completion + "\nfi\n"
	return script + s.grants() + s.portsEnv(), nil
}

// invariant: in a VM the shell sources the environment files of the
// programs granted to it by their path, so the script itself never holds a
// token; a Mac has none.
func (s *Block) grants() string {
	if s.o.ThisMachine != catalog.Linux && s.o.ThisMachine != catalog.MacOS {
		return ""
	}
	return "\nfor f in \"$HOME/" + catalog.GrantsDir + "/\"*.env(N); do source \"$f\"; done\n"
}

// invariant: in a VM the profiles' exports the guest wrote are a file read, never
// a subprocess: a shell start never waits on the host.
func (s *Block) portsEnv() string {
	if (s.o.ThisMachine != catalog.Linux && s.o.ThisMachine != catalog.MacOS) || s.o.PortsEnv == "" {
		return ""
	}
	data, err := s.files.ReadFile(s.o.PortsEnv)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "\n" + string(data)
}
