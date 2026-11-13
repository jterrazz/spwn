package macos

import (
	"context"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func (m *Mac) observeRemoteLogin(ctx context.Context, want []settings.KeyValue) settings.Observation {
	return judged(m.remoteLoginValues(ctx), want, "Remote Login on", "")
}

func (m *Mac) applyRemoteLogin(ctx context.Context) (settings.Change, error) {
	return m.remoteLogin(ctx, "on")
}

func (m *Mac) undoRemoteLogin(ctx context.Context) (settings.Change, error) {
	return m.remoteLogin(ctx, "off")
}

// invariant: a failing systemsetup is an error: the step never reads as done when
// Remote Login did not change.
func (m *Mac) remoteLogin(ctx context.Context, state string) (settings.Change, error) {
	change := settings.Change{Before: m.remoteLoginValues(ctx)}
	if _, err := m.must(ctx, "systemsetup", "-setremotelogin", state); err != nil {
		return change, err
	}
	change.Steps = []settings.ChangeStep{done("Remote Login", state)}
	change.After = m.remoteLoginValues(ctx)
	return change, nil
}

// outside: the check runs as the user, and `systemsetup -getremotelogin` then exits 0
// with its error on stdout; the system domain's overrides are readable without root
// and carry the sshd state systemsetup sets.
func (m *Mac) remoteLoginValues(ctx context.Context) []settings.KeyValue {
	now := "off"
	if m.o.Jobs.Overrides(ctx, backgroundcontract.Domain{System: true})["com.openssh.sshd"] {
		now = "on"
	}
	return []settings.KeyValue{{Key: "Remote Login", Value: now}}
}
