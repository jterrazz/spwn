package npm

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const command = "npm"

// value: the variable that moves npm's global prefix, what `-g` reads and
// removes, away from the node that runs it.
const (
	PrefixVar = "NPM_CONFIG_PREFIX"
)

type Npm struct {
	run    local.Runner
	bundle contract.Bundler
}

func New(run local.Runner, bundle contract.Bundler) *Npm { return &Npm{run: run, bundle: bundle} }

func (n *Npm) Method() catalog.Method { return catalog.MethodNpm }

func (n *Npm) Manager() contract.Manager {
	return contract.Manager{Name: command, App: "node"}
}

func (n *Npm) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (n *Npm) Ready(context.Context) error {
	if _, ok := n.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (n *Npm) Inventory(ctx context.Context) (map[string]string, error) {
	// outside: `npm ls -g` exits 1 on an extraneous or missing peer while its
	// JSON still lists every global package, so its exit code is not read.
	if err := n.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := n.run.Output(ctx, local.Command(command, "ls", "-g", "--depth=0", "--json"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &doc); err != nil {
		return nil, err
	}
	list := map[string]string{}
	for name, d := range doc.Dependencies {
		if !bundled(name) {
			list[name] = d.Version
		}
	}
	return list, nil
}

// outside: node's own install lays npm and corepack in its global prefix,
// so `npm ls -g` lists them beside what was installed there.
func bundled(name string) bool { return name == command || name == "corepack" }

func (n *Npm) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (n *Npm) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return n.bundle.Bundle(ctx, n.lines(installs), false, logs)
}

func (n *Npm) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return n.bundle.Bundle(ctx, n.lines(installs), true, logs)
}

func (n *Npm) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.NpmInstall)
		if !ok {
			return errors.New(i.Key() + " is not an npm package")
		}
		c := local.Command(command, "uninstall", "-g", v.Package)
		c.Logs = logs
		if err := local.Attach(ctx, n.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (n *Npm) lines(installs []catalog.Install) []string {
	out := make([]string, 0, len(installs))
	for _, i := range installs {
		out = append(out, n.Line(i))
	}
	return out
}

func (n *Npm) Line(i catalog.Install) string {
	v, ok := i.(catalog.NpmInstall)
	if !ok {
		return ""
	}
	return `npm "` + v.Package + `"`
}
