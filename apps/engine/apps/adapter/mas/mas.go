package mas

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
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
	return contract.Manager{Name: command, App: command}
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

func (m *Mas) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: an app is behind when `mas outdated` lists its id, with the
	// version it would bring; a mas that does not answer in the budget (an App
	// Store signed out) leaves every key unknown.
	if len(keys) == 0 {
		return map[string]apps.Currency{}, nil
	}
	out, err := m.run.Output(ctx, local.Cmd{Name: command, Args: []string{"outdated"}, Timeout: contract.CurrencyBudget})
	if err == nil && out.ExitCode == 0 {
		return apps.Verdicts(keys, parseOutdated(out.Stdout)), nil
	}
	return apps.Unread(keys), nil
}

func parseOutdated(stdout string) map[string]string {
	// outside: mas prints `<id> <name> (<installed> -> <newest>)` a line.
	behind := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if _, err := strconv.Atoi(fields[0]); err != nil {
			continue
		}
		if arrow := strings.LastIndex(line, "-> "); arrow >= 0 {
			behind[fields[0]] = strings.TrimSuffix(strings.TrimSpace(line[arrow+3:]), ")")
		}
	}
	return behind
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
