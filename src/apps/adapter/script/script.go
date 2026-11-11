package script

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
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
	// value: the run has a terminal the owner answers on.
	terminal bool
}

func New(run local.Runner, terminal bool) *Script { return &Script{run: run, terminal: terminal} }

func (s *Script) env(v catalog.ScriptInstall) []string {
	if s.terminal {
		return append(slices.Clone(v.Env), v.OnTerminal...)
	}
	return append(slices.Clone(v.Env), v.OffTerminal...)
}

func (s *Script) Method() catalog.Method    { return catalog.MethodScript }
func (s *Script) Manager() contract.Manager { return contract.Manager{} }

func (s *Script) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, UpdateAll: true}
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

func (s *Script) Remove(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}
