package npm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
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
	where  func(ctx context.Context, runtime string) ([]string, error)
}

func New(run local.Runner, bundle contract.Bundler, where func(ctx context.Context, runtime string) ([]string, error)) *Npm {
	// value: where names the folders of every global version of a runtime
	// its manager holds, none when it holds none; nil reads npm's own prefix.
	return &Npm{run: run, bundle: bundle, where: where}
}

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
	if err := n.Ready(ctx); err != nil {
		return nil, err
	}
	folders, err := n.global(ctx)
	if err != nil {
		return nil, err
	}
	list := map[string]string{}
	for _, folder := range folders {
		listed, err := n.list(ctx, folder)
		if err != nil {
			return nil, err
		}
		for name, version := range listed {
			if _, seen := list[name]; !seen {
				list[name] = version
			}
		}
	}
	return list, nil
}

func (n *Npm) list(ctx context.Context, folder []string) (map[string]string, error) {
	// outside: `npm ls -g` exits 1 on an extraneous or missing peer while its
	// JSON still lists every global package, so its exit code is not read.
	out, err := n.run.Output(ctx, local.Command(command, append([]string{"ls", "-g", "--depth=0", "--json"}, folder...)...))
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

func (n *Npm) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: a package is behind when any node its manager holds globally
	// has it at another version than the registry's latest; one folder whose
	// read did not answer leaves every key unknown, as it does the list.
	if len(keys) == 0 {
		return map[string]apps.Currency{}, nil
	}
	folders, err := n.global(ctx)
	if err == nil {
		return n.verdicts(ctx, keys, folders), nil
	}
	return apps.Unread(keys), nil
}

func (n *Npm) verdicts(ctx context.Context, keys []string, folders [][]string) map[string]apps.Currency {
	behind := map[string]string{}
	for _, folder := range folders {
		found, ok := n.outdated(ctx, folder)
		if !ok {
			return apps.Unread(keys)
		}
		for name, latest := range found {
			if _, seen := behind[name]; !seen {
				behind[name] = latest
			}
		}
	}
	return apps.Verdicts(keys, behind)
}

func (n *Npm) outdated(ctx context.Context, folder []string) (map[string]string, bool) {
	// outside: `npm outdated` exits 1 when it lists a package, 0 when it lists
	// none (printing nothing), and prints nothing on stdout when the registry
	// did not answer: an empty stdout is no answer at exit 1 only.
	out, err := n.run.Output(ctx, local.Cmd{Name: command, Args: append([]string{"outdated", "-g", "--json"}, folder...), Timeout: contract.CurrencyBudget})
	if err != nil || out.ExitCode > 1 || out.ExitCode == 1 && strings.TrimSpace(out.Stdout) == "" {
		return nil, false
	}
	if strings.TrimSpace(out.Stdout) == "" {
		return map[string]string{}, true
	}
	var doc map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &doc); err != nil {
		return nil, false
	}
	if _, failed := doc["error"]; failed {
		return nil, false
	}
	behind := map[string]string{}
	for name, p := range doc {
		if p.Latest != "" && p.Latest != p.Current {
			behind[name] = p.Latest
		}
	}
	return behind, true
}

func (n *Npm) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return n.bundle.Bundle(ctx, n.lines(installs), false, logs)
}

func (n *Npm) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return n.bundle.Bundle(ctx, n.lines(installs), true, logs)
}

func (n *Npm) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	folders, err := n.global(ctx)
	if err != nil {
		return err
	}
	for _, i := range installs {
		v, ok := i.(catalog.NpmInstall)
		if !ok {
			return errors.New(i.Key() + " is not an npm package")
		}
		for _, folder := range folders {
			c := local.Command(command, append(append([]string{"uninstall", "-g"}, folder...), v.Package)...)
			c.Logs = logs
			if err := local.Attach(ctx, n.run, c); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Npm) global(ctx context.Context) ([][]string, error) {
	// invariant: what is read and what is removed are the packages of every
	// node its manager holds globally, all of which that manager's removal
	// takes back, each named by its folder, so no PATH, folder or variable
	// picks another and no version of them goes unread; with none, npm's
	// own prefix.
	if n.where == nil {
		return [][]string{nil}, nil
	}
	folders, err := n.where(ctx, n.Manager().App)
	if err != nil {
		return nil, err
	}
	if len(folders) == 0 {
		return [][]string{nil}, nil
	}
	out := make([][]string, 0, len(folders))
	for _, f := range folders {
		out = append(out, []string{"--prefix", f})
	}
	return out, nil
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
