package wire

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/src/machinefile/adapter/git"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	machinefilecontract "github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/surface/cli/explain"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// value: the owner's files, ~/.config/station/{catalog,machinefiles}; in
// a world, its Mac's station files.
func (m machine) userSource() machinefilecontract.Source {
	if m.world != nil {
		return m.world.userSource()
	}
	return files.New(m.sys.FS, filepath.Join(m.paths.Prefs, "catalog"), filepath.Join(m.paths.Prefs, "machinefiles"), machinefile.User, "~/.config/station")
}

func builtinSource() machinefilecontract.Source {
	return files.New(files.Embedded{FS: catalog.BuiltinFiles()}, "items", "machinefiles", machinefile.Builtin, "built-in")
}

// invariant: the sources strongest first: the owner's, the shared source
// preferences.json names (read from its clone in the cache), the
// built-in.
func (m machine) machinefileSources() []machinefilecontract.Source {
	sources := []machinefilecontract.Source{m.userSource()}
	if shared, ok, err := machinefilecontract.NewSourcePrefs(m.stores.prefs).Get(); err == nil && ok {
		if m.world != nil {
			sources = append(sources, m.world.sharedSource(shared))
		} else {
			sources = append(sources, git.New(m.sys.Run, m.sys.FS, m.paths.Cache, shared))
		}
	}
	return append(sources, builtinSource())
}

// invariant: the catalog every command reads is the owner's items over
// the built-in ones, a same-name item replacing it whole; a file of the
// owner's the strict decoder refuses stops the command with its line.
func (m *machine) layerCatalog() error {
	user, err := m.userSource().Read(context.Background())
	if err != nil {
		return err
	}
	if len(user.Files) == 0 {
		return nil
	}
	builtin, err := builtinSource().Read(context.Background())
	if err != nil {
		return err
	}
	c, err := machinefile.CatalogOf([]machinefile.Tree{user, builtin})
	if err != nil {
		return machinefilecalls.Worded(err)
	}
	m.cat = c
	return nil
}

func (m machine) machinefileDeps() machinefilecalls.Deps {
	return machinefilecalls.Deps{Sources: m.machinefileSources, Machine: machineOf(m.profile), Name: m.profile.Ref.Name}
}

// value: this machine as a Machinefile's selector reads it: Station's
// Linux machines are Ubuntu.
func machineOf(p coremachine.Profile) machinefile.Machine {
	os := catalog.OS(p.System.OS)
	if p.System.OS == coremachine.Linux {
		os = catalog.OSUbuntu
	}
	return machinefile.Machine{Kind: catalog.MachineRole(p.Kind), System: catalog.SystemSpec{OS: os, Version: p.System.Version, Arch: catalog.Arch(p.System.Arch)}}
}

func explainCommand(env kit.Env, m machine) *cobra.Command {
	return explain.Command(explain.Deps{ThisMachine: m.where}, env)
}
