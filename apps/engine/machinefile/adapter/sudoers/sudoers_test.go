package sudoers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/sudoers"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

func TestSudoKeepsTheContract(t *testing.T) {
	contracttest.Password(t, func(t *testing.T, asks bool) contract.Password {
		r := localtest.NewRunner(t)
		code := 0
		if asks {
			code = 1
		}
		r.On("sudo", "-k", "-n", "true").Reply(local.Output{ExitCode: code, Stderr: "sudo: a password is required"})
		return sudoers.New(r)
	})
}

func TestSudoThatCannotBeRunCannotTell(t *testing.T) {
	r := localtest.NewRunner(t)
	r.On("sudo", "-k", "-n", "true").Fail(local.ErrNotFound)
	if asked, err := sudoers.New(r).Asked(context.Background()); !asked || !errors.Is(err, local.ErrNotFound) {
		t.Errorf("Asked = %v, %v; want asked, with sudo's error", asked, err)
	}
}

func TestSudoIsAskedNothingAndBounded(t *testing.T) {
	// invariant: the read never prompts and never waits unbounded.
	r := localtest.NewRunner(t)
	r.On("sudo", "-k", "-n", "true").Reply(local.Output{})
	if _, err := sudoers.New(r).Asked(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.Calls) != 1 || r.Calls[0].Timeout <= 0 || r.Calls[0].Stdin != "" {
		t.Errorf("calls %+v; want one, bounded, with nothing on its stdin", r.Calls)
	}
}
