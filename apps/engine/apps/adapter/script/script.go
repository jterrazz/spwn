package script

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// hazard: an installer whose words go to its log cannot be seen asking, so it
// reads one RETURN on stdin: the consent the owner gave by asking the install.
// A password it asks is sudo's, read on /dev/tty, never on stdin; Homebrew's
// install.sh turns NONINTERACTIVE and `sudo -n` on a stdin that is no TTY, so
// its catalogue line sets INTERACTIVE where the run has a terminal, and
// NONINTERACTIVE where it has none (INTERACTIVE with CI set aborts it). Its
// form stays `bash -c "$(curl ...)"`, the script never read from stdin.
const curlInstaller = `script="$(curl -fsSL "$0")" && exec bash -c "$script" bash "$@"`

type Script struct {
	run local.Runner
	// value: the home a value's `~/` opens at.
	home string
	// value: the run has a terminal the owner answers on.
	terminal bool
}

func New(run local.Runner, home string, terminal bool) *Script {
	return &Script{run: run, home: home, terminal: terminal}
}

// invariant: a variable's value that opens with `~/` is under the home,
// since an installer reads it as is (`HELM_INSTALL_DIR=~/.local/bin`).
func (s *Script) env(v catalog.ScriptInstall) []string {
	pairs := append(slices.Clone(v.Env), v.OffTerminal...)
	if s.terminal {
		pairs = append(slices.Clone(v.Env), v.OnTerminal...)
	}
	for i, kv := range pairs {
		k, v, _ := strings.Cut(kv, "=")
		if rest, ok := strings.CutPrefix(v, "~/"); ok {
			pairs[i] = k + "=" + path.Join(s.home, rest)
		}
	}
	return pairs
}

func (s *Script) Method() catalog.Method    { return catalog.MethodScript }
func (s *Script) Manager() contract.Manager { return contract.Manager{} }

func (s *Script) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true}
}

func (s *Script) Ready(context.Context) error { return nil }

func (s *Script) Inventory(context.Context) (map[string]string, error) {
	return nil, contract.ErrUnlisted
}

func (s *Script) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (s *Script) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.ScriptInstall)
		if !ok {
			return errors.New(i.Key() + " is not an install script")
		}
		c := local.Command("bash", append([]string{"-c", curlInstaller, v.URL}, v.Args...)...)
		c.Env, c.Logs = s.env(v), logs
		if len(logs) > 0 {
			c.Stdin = "\n"
		}
		if err := local.Attach(ctx, s.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Script) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.ScriptInstall)
		if !ok || v.Upgrade == "" {
			if err := s.Install(ctx, []catalog.Install{i}, logs); err != nil {
				return err
			}
			continue
		}
		c := local.Command("bash", "-c", v.Upgrade)
		c.Env, c.Logs = s.env(v), logs
		if err := local.Attach(ctx, s.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Script) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	// invariant: an installer is taken back by its `undo` alone, run as the
	// upgrade is, with the install's variables; one with none is refused
	// before anything runs, for every install of the call.
	scripts := make([]catalog.ScriptInstall, 0, len(installs))
	for _, i := range installs {
		v, ok := i.(catalog.ScriptInstall)
		if !ok {
			return errors.New(i.Key() + " is not an install script")
		}
		if v.Undo == "" {
			return contract.ErrCannot
		}
		scripts = append(scripts, v)
	}
	for _, v := range scripts {
		c := local.Command("bash", "-c", v.Undo)
		c.Env, c.Logs = s.env(v), logs
		if err := local.Attach(ctx, s.run, c); err != nil {
			return err
		}
	}
	return nil
}
