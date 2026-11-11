package wire

import (
	"context"
	"errors"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/src/machinefile/adapter/git"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	machinefilecontract "github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/output"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
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
// owner's the strict decoder refuses leaves the built-in catalog, and is
// returned worded at its line for the gate to judge.
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

// value: the owner's file refused, as Needs you says it: the worded
// problem at its file and line, and the file to fix (the owner's catalog
// folder when the refusal names none).
func refusalOf(broken error) *overviewcalls.Refusal {
	if broken == nil {
		return nil
	}
	file := "~/.config/station/catalog"
	var named *catalog.FileError
	if errors.As(broken, &named) {
		file = named.File
	}
	return &overviewcalls.Refusal{Text: kit.Translate(broken, "").Message, File: file}
}

// value: the commands that read no catalog, by their word under the root:
// help, the shell's completion, the engine (the menu bar's, which must
// keep answering whatever file the owner broke, and its start and stop)
// and a world's own events.
func catalogFree() []string {
	return []string{"help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd, "engine", "start", "stop", "world"}
}

// invariant: an owner's item file refused stops every command that reads
// the catalog, at its file and line; a command that reads none runs on
// the built-in items, and so do --version and --help, which cobra answers
// before any command runs: each says the file is refused on stderr, once.
// The returned func says it after the run when nothing else did.
func gateCatalog(root *cobra.Command, out *output.Writer, broken error) (reported func()) {
	if broken == nil {
		return func() {}
	}
	worded := kit.Translate(broken, "")
	said := false
	say := func() {
		if !said {
			said = true
			out.Note(worded.Message, "this run reads the built-in items")
		}
	}
	pre := root.PersistentPreRunE
	root.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		top := c
		for top.HasParent() && top.Parent() != root {
			top = top.Parent()
		}
		if top == root || !slices.Contains(catalogFree(), top.Name()) {
			said = true
			return worded
		}
		say()
		if pre == nil {
			return nil
		}
		return pre(c, args)
	}
	return say
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
