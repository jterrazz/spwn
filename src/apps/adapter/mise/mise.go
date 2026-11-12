package mise

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const command = "mise"

type Mise struct{ run local.Runner }

func New(run local.Runner) *Mise { return &Mise{run: run} }

func (m *Mise) Method() catalog.Method { return catalog.MethodMise }

func (m *Mise) Manager() contract.Manager {
	return contract.Manager{Name: command, App: command}
}

func (m *Mise) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (m *Mise) Ready(context.Context) error {
	if _, ok := m.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (m *Mise) Inventory(ctx context.Context) (map[string]string, error) {
	held, err := m.global(ctx)
	if err != nil {
		return nil, err
	}
	list := map[string]string{}
	for name, versions := range held {
		list[name] = versions[0].Version
	}
	return list, nil
}

func (m *Mise) Where(ctx context.Context, runtime string) ([]string, error) {
	// value: the folders of every version of runtime mise holds globally,
	// all of which `unuse --global` takes back; none when mise is missing
	// or holds none.
	if _, ok := m.run.LookPath(command); !ok {
		return nil, nil
	}
	held, err := m.global(ctx)
	if err != nil {
		return nil, err
	}
	var folders []string
	for _, v := range held[runtime] {
		folders = append(folders, v.Path)
	}
	return folders, nil
}

type version struct {
	Version   string `json:"version"`
	Path      string `json:"install_path"`
	Installed bool   `json:"installed"`
}

// value: every installed version of each tool the global file names, in
// the order mise lists them.
func (m *Mise) global(ctx context.Context) (map[string][]version, error) {
	if err := m.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := m.run.Output(ctx, local.Command(command, "ls", "--global", "--json"))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("mise ls exited " + strconv.Itoa(out.ExitCode))
	}
	var doc map[string][]version
	if err := json.Unmarshal([]byte(out.Stdout), &doc); err != nil {
		return nil, err
	}
	held := map[string][]version{}
	for name, versions := range doc {
		for _, v := range versions {
			if v.Installed {
				held[name] = append(held[name], v)
			}
		}
	}
	return held, nil
}

func (m *Mise) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: mise lists a runtime only where a newer version satisfies what
	// the global file requests, so a version pinned there exactly is never
	// behind: its pin is the repo's to move (ADR-003). A read that did not
	// answer leaves every key unknown.
	if len(keys) == 0 {
		return map[string]apps.Currency{}, nil
	}
	behind, ok := m.outdated(ctx, keys)
	if !ok {
		return apps.Unread(keys), nil
	}
	return apps.Verdicts(keys, behind), nil
}

func (m *Mise) outdated(ctx context.Context, keys []string) (map[string]string, bool) {
	out, err := m.run.Output(ctx, local.Cmd{Name: command, Args: append([]string{"outdated", "--json", "--"}, keys...), Timeout: contract.CurrencyBudget})
	if err != nil || out.ExitCode != 0 {
		return nil, false
	}
	var doc map[string]struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &doc); err != nil {
		return nil, false
	}
	behind := map[string]string{}
	for name, runtime := range doc {
		if runtime.Latest != "" {
			behind[name] = runtime.Latest
		}
	}
	return behind, true
}

func (m *Mise) Newest(ctx context.Context, in catalog.Install) (string, error) {
	v, ok := in.(catalog.MiseInstall)
	if !ok {
		return "", errors.New(in.Key() + " is not a mise runtime")
	}
	if err := m.Ready(ctx); err != nil {
		return "", err
	}
	out, err := m.run.Output(ctx, local.Cmd{Name: command, Args: []string{"latest", "--", v.App}, Timeout: contract.CurrencyBudget})
	if err != nil {
		return "", err
	}
	if out.ExitCode != 0 {
		return "", errors.New("mise latest exited " + strconv.Itoa(out.ExitCode))
	}
	return strings.TrimSpace(out.Stdout), nil
}

func (m *Mise) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return m.each(ctx, installs, logs, func(v catalog.MiseInstall) local.Cmd {
		return local.Command(command, "use", "--global", v.App+"@"+v.Version)
	})
}

func (m *Mise) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return m.each(ctx, installs, logs, func(v catalog.MiseInstall) local.Cmd { return local.Command(command, "upgrade", v.App) })
}

func (m *Mise) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return m.each(ctx, installs, logs, func(v catalog.MiseInstall) local.Cmd {
		return local.Command(command, "unuse", "--global", "--no-prune", v.App)
	})
}

func (m *Mise) each(ctx context.Context, installs []catalog.Install, logs contract.Logs, argv func(catalog.MiseInstall) local.Cmd) error {
	for _, i := range installs {
		v, ok := i.(catalog.MiseInstall)
		if !ok {
			return errors.New(i.Key() + " is not a mise runtime")
		}
		c := argv(v)
		c.Logs = logs
		if err := local.Attach(ctx, m.run, c); err != nil {
			return err
		}
	}
	return nil
}

func Pinned(home string) []string {
	// invariant: mise reads no config at or above home but its global one, the
	// file `use --global` and `unuse --global` write, so a shim answers with the
	// global version; and no shim nor `mise x` installs a missing runtime by
	// itself, so a read installs nothing.
	return []string{"MISE_CEILING_PATHS=" + home, "MISE_AUTO_INSTALL=false", "MISE_EXEC_AUTO_INSTALL=false", "MISE_NOT_FOUND_AUTO_INSTALL=false"}
}

func Environment(home string, vars map[string]string, moved ...string) []string {
	// invariant: a manager command answers for the versions mise holds
	// globally, whatever shell station was started from: mise's shims come
	// first on its PATH, and no runtime's own folder is on it (mise's
	// installs, put there by `mise activate`; nvm's and asdf's versions),
	// since a project's node found there lists what that node holds while
	// `mise unuse --global node` takes back the global one; mise's profile,
	// a version named in the environment, `mise activate`'s session and the
	// moved variables another manager names (npm's prefix) are left out,
	// since each moves what is read away from what is removed.
	data := dataDir(home, vars)
	shims := path.Join(data, "shims")
	versions := []string{path.Join(data, "installs"), path.Join(or(vars["NVM_DIR"], path.Join(home, ".nvm")), "versions"), path.Join(or(vars["ASDF_DATA_DIR"], path.Join(home, ".asdf")), "installs")}
	search := []string{shims}
	for _, dir := range strings.Split(vars["PATH"], ":") {
		if dir != "" && !slices.Contains(search, dir) && !slices.ContainsFunc(versions, func(v string) bool { return strings.HasPrefix(dir, v+"/") }) {
			search = append(search, dir)
		}
	}
	pins := Pinned(home)
	env := []string{"PATH=" + strings.Join(search, ":")}
	for k, v := range vars {
		set := func(kv string) bool { return strings.HasPrefix(kv, k+"=") }
		if k != "PATH" && !moves(k) && !slices.ContainsFunc(pins, set) && !slices.ContainsFunc(moved, func(m string) bool { return strings.EqualFold(k, m) }) {
			env = append(env, k+"="+v)
		}
	}
	slices.Sort(env)
	return append(env, pins...)
}

func Kept(home string, vars map[string]string) map[catalog.Method][]string {
	// value: the folders mise keeps its runtimes and their shims in, by
	// its method: a command found there for an app of mise's is mise's,
	// which its own listing reads.
	data := dataDir(home, vars)
	return map[catalog.Method][]string{catalog.MethodMise: {path.Join(data, "installs"), path.Join(data, "shims")}}
}

func dataDir(home string, vars map[string]string) string {
	if data := vars["MISE_DATA_DIR"]; data != "" {
		return data
	}
	return path.Join(home, ".local", "share", command)
}

func moves(name string) bool {
	switch {
	case name == "MISE_ENV", name == "MISE_PROFILE", name == "MISE_SHELL":
		return true
	case strings.HasPrefix(name, "__MISE_"):
		return true
	}
	return strings.HasPrefix(name, "MISE_") && strings.HasSuffix(name, "_VERSION")
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
