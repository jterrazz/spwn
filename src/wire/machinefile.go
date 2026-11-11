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
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/surface/cli/explain"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// value: the owner's files, ~/.config/station/{catalog,machinefiles}; in
// a world, its Mac's station files.
func userSource(world *worldMac, fsys local.FS, prefs string) machinefilecontract.Source {
	if world != nil {
		return world.userSource()
	}
	return files.New(fsys, machinefilecontract.FileLinks, filepath.Join(prefs, "catalog"), filepath.Join(prefs, "machinefiles"), machinefile.User, "~/.config/station")
}

func (m machine) userSource() machinefilecontract.Source {
	return userSource(m.world, m.sys.FS, m.paths.Prefs)
}

func builtinSource() machinefilecontract.Source {
	return files.New(files.Embedded{FS: catalog.BuiltinFiles()}, machinefilecontract.NoLinks, "items", "machinefiles", machinefile.Builtin, "built-in")
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
// the built-in ones, a same-name item replacing it whole. A file of the
// owner's the strict decoder refuses is returned worded at its line, and
// stops the command.
func layerCatalog(user machinefilecontract.Source, builtin catalog.Catalog) (catalog.Catalog, error) {
	tree, err := user.Read(context.Background())
	if err != nil {
		return builtin, err
	}
	if len(tree.Files) == 0 {
		return builtin, nil
	}
	embedded, err := builtinSource().Read(context.Background())
	if err != nil {
		return builtin, err
	}
	c, err := machinefile.CatalogOf([]machinefile.Tree{tree, embedded})
	if err != nil {
		return builtin, machinefilecalls.Worded(err)
	}
	return c, nil
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
