package mas

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const command = "mas"

// invariant: signed in, the App Store queues a download within seconds and mas
// says "Downloading"; signed out, it waits on a sign-in no terminal can give
// and mas says nothing. The bound leaves room for sudo's password.
const answer = 60 * time.Second

type Mas struct {
	run    local.Runner
	answer time.Duration
}

func New(run local.Runner) *Mas { return &Mas{run: run, answer: answer} }

func (m *Mas) Method() catalog.Method { return catalog.MethodMas }

func (m *Mas) Manager() contract.Manager {
	return contract.Manager{Name: command, App: command, Needs: "homebrew"}
}

func (m *Mas) Can() contract.Abilities {
	// invariant: the App Store updates its own apps, so a bare update leaves
	// them; one is updated by name.
	return contract.Abilities{Install: true, Update: true, Remove: true}
}

func (m *Mas) Ready(context.Context) error {
	if _, ok := m.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (m *Mas) Inventory(ctx context.Context) (map[string]string, error) {
	if err := m.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := m.run.Output(ctx, local.Command(command, "list"))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("mas list exited " + strconv.Itoa(out.ExitCode))
	}
	return parseList(out.Stdout), nil
}

func (m *Mas) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (m *Mas) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	// invariant: an App Store app is installed by mas itself, not by the shared
	// brew bundle: inside the bundle mas's output is held until it ends, so a
	// signed-out App Store would hang the whole bundle with no word.
	return m.each(ctx, "install", installs, logs)
}

func (m *Mas) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return m.each(ctx, "update", installs, logs)
}

func (m *Mas) each(ctx context.Context, verb string, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.MasInstall)
		if !ok {
			return errors.New(i.Key() + " is not an App Store app")
		}
		c := local.Command(command, verb, strconv.Itoa(v.ID))
		c.Answer, c.Logs = m.answer, logs
		err := local.Attach(ctx, m.run, c)
		if errors.Is(err, local.ErrSilent) {
			return &apps.SignedOutError{Store: "the App Store", Waited: m.answer, Hint: "open the App Store and sign in"}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mas) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.MasInstall)
		if !ok {
			return errors.New(i.Key() + " is not an App Store app")
		}
		c := local.Command(command, "uninstall", strconv.Itoa(v.ID))
		c.Logs = logs
		if err := local.Attach(ctx, m.run, c); err != nil {
			return err
		}
	}
	return nil
}

func parseList(stdout string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		version := ""
		if open := strings.LastIndex(line, "("); open >= 0 {
			version = strings.TrimSuffix(strings.TrimSpace(line[open+1:]), ")")
		}
		out[fields[0]] = version
	}
	return out
}
