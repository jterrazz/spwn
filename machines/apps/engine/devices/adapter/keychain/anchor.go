package keychain

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const (
	command = "security"
	// value: the login keychain's item that holds the book's head, under a
	// name apart from the vault's "station" items so the vault never lists it.
	itemName = "station-devices"
	account  = "head"
	// outside: `security` exits 44 when the item is not in the keychain.
	notFound = 44
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Anchor struct{ run Runner }

var _ contract.Anchor = Anchor{}

func New(run Runner) Anchor { return Anchor{run: run} }

type UnreadableError struct {
	// value: why the login keychain did not give the book's head.
	Cause string
}

func (e *UnreadableError) Error() string {
	return "the device book's head is not readable in the login keychain: " + e.Cause
}

func cause(out local.Output, err error) string {
	if err != nil {
		return err.Error()
	}
	return strings.TrimSpace(out.Stderr)
}

func (a Anchor) Read(ctx context.Context) (devices.Head, error) {
	out, err := a.run.Output(ctx, local.Cmd{Name: command, Args: []string{"find-generic-password", "-s", itemName, "-a", account, "-w"}})
	switch {
	case err != nil:
		return devices.Head{}, &UnreadableError{Cause: cause(out, err)}
	case out.ExitCode == notFound:
		return devices.Head{}, nil
	case out.ExitCode != 0:
		return devices.Head{}, &UnreadableError{Cause: cause(out, nil)}
	}
	h, err := devices.ParseHead(out.Stdout)
	if err != nil {
		return devices.Head{}, &UnreadableError{Cause: err.Error()}
	}
	return h, nil
}

var errUnwritable = errors.New("the login keychain refused the device book's head")

func (a Anchor) Write(ctx context.Context, h devices.Head) error {
	// invariant: the head travels on security's stdin, in its interactive
	// mode, as the vault's secrets do; -U replaces the item in place.
	line := "add-generic-password -U -s " + itemName + " -a " + account + " -w \"" + h.String() + "\"\n"
	out, err := a.run.Output(ctx, local.Cmd{Name: command, Args: []string{"-i"}, Stdin: line})
	if err != nil || out.ExitCode != 0 {
		return errors.Join(errUnwritable, errors.New(cause(out, err)))
	}
	return nil
}
