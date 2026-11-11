package mise

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strconv"

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
	var doc map[string][]struct {
		Version   string `json:"version"`
		Installed bool   `json:"installed"`
	}
	if err := json.Unmarshal([]byte(out.Stdout), &doc); err != nil {
		return nil, err
	}
	list := map[string]string{}
	for name, versions := range doc {
		for _, v := range versions {
			if v.Installed {
				list[name] = v.Version
				break
			}
		}
	}
	return list, nil
}

func (m *Mise) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
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
	return m.each(ctx, installs, logs, func(v catalog.MiseInstall) local.Cmd { return local.Command(command, "unuse", "--global", v.App) })
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
	// global version.
	return []string{"MISE_CEILING_PATHS=" + home}
}

func (m *Mise) ShimsFirst(home string, vars map[string]string) []string {
	// invariant: a bundle runs with mise's shims first once mise is installed,
	// so a formula's post-install finds the runtimes it pins; the shims live in
	// $MISE_DATA_DIR, else ~/.local/share/mise.
	if _, ok := m.run.LookPath(command); !ok {
		return nil
	}
	data := vars["MISE_DATA_DIR"]
	if data == "" {
		data = path.Join(home, ".local", "share", command)
	}
	return []string{"PATH=" + path.Join(data, "shims") + ":" + vars["PATH"]}
}
