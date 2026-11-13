package wire

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/studio"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	accountscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/accounts"
	accountsshare "github.com/jterrazz/studio-station/apps/engine/surface/cli/accounts/share"
	accountstakeback "github.com/jterrazz/studio-station/apps/engine/surface/cli/accounts/takeback"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func accountsCommand(env kit.Env, m machine) *cobra.Command {
	return accountscmd.Command(accountscmd.Deps{ThisMachine: m.where}, env,
		accountsshare.Command(accountsshare.Deps{Now: m.sys.Clock.Now}, env),
		accountstakeback.Command(env),
	)
}

// invariant: station secrets and its verbs are the hidden v2 words of
// station accounts for one release (SPEC §4.3); allow and disallow are
// refused, the share list being the owner's to write.
func secretsAlias(env kit.Env, m machine) *cobra.Command {
	return kit.Alias("secrets", accountscmd.Command(accountscmd.Deps{ThisMachine: m.where}, env,
		kit.AliasArgs(kit.Alias("lend", accountsshare.LendAlias(accountsshare.Deps{Now: m.sys.Clock.Now}, env), "share"), "<account>"),
		kit.AliasArgs(kit.Alias("recall", accountstakeback.RecallAlias(env), "take-back"), "<account>"),
		accountscmd.Retired("allow", env),
		accountscmd.Retired("disallow", env),
	), "accounts")
}

func (m machine) secretStores() accountscalls.SharingStores {
	return accountscalls.SharingStores{
		ShareList: m.stores.shareList(), Macs: m.stores.macs(), Name: m.stores.name(),
		Shares: m.stores.shares, Machines: m.stores.machines,
	}
}

// value: whether a VM of this Mac runs, as its engine says, asked through
// the engine's socket.
func (m machine) vmRunning(ctx context.Context, vm string) (bool, error) {
	live, err := m.macEngine().VMs(ctx)
	if err != nil {
		return false, err
	}
	return live[vm].State == engine.StateRunning, nil
}

func receives(guest localGuest) func(context.Context, string) (bool, error) {
	// value: a VM whose station is older than its kind's answer receives.
	ask := unix.New(sharedGuest{guest}, nil)
	return func(ctx context.Context, vm string) (bool, error) {
		kind, err := ask.KindOf(ctx, vm)
		if err != nil || kind == "" {
			return err == nil, err
		}
		return accounts.SharedInto(coremachine.Profile{Ref: coremachine.Ref{Name: vm}, Kind: kind}).OK, nil
	}
}

// value: a command inside a VM as the engine runs it: the socket's, or
// the engine's own when it is the engine that shares.
type localGuest interface {
	GuestExec(ctx context.Context, vm, line, stdin string) (engine.GuestRan, error)
}

type sharedGuest struct{ g localGuest }

func (s sharedGuest) GuestExec(ctx context.Context, vm, line, stdin string) (stdout string, code int, err error) {
	out, err := s.g.GuestExec(ctx, vm, line, stdin)
	return out.Stdout, out.Code, err
}

func (m machine) secrets() *accountscalls.Sharing { return m.secretsVia(m.macEngine(), m.vmRunning) }

func (m machine) secretsVia(guest localGuest, running func(context.Context, string) (bool, error)) *accountscalls.Sharing {
	return accountscalls.NewSharing(sharingGuest{guest}, m.secretStores(), m.secretSource(studio.Options{Budget: accountscalls.ExportBudget}), m.identities(engine.ChildrenDir(m.paths.State)), accountscalls.SharingOptions{
		ThisMachine: m.kind, Running: running, Receives: receives(guest), Now: m.sys.Clock.Now,
		Keep: m.keepIn(filepath.Join(m.paths.State, "logs", "accounts.log")),
	})
}

type sharingGuest struct{ g localGuest }

func (s sharingGuest) GuestExec(ctx context.Context, vm, line, stdin string) (accountscalls.GuestRun, error) {
	out, err := s.g.GuestExec(ctx, vm, line, stdin)
	return accountscalls.GuestRun{Stdout: out.Stdout, Stderr: out.Stderr, ExitCode: out.Code}, err
}

// value: appends a line to the log file, and answers where it is kept,
// shown from home; empty when it could not be kept.
func (m machine) keepIn(log string) func(line string) string {
	return func(line string) string {
		before, err := m.sys.FS.ReadFile(log)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return ""
		}
		if err := m.sys.FS.MkdirAll(filepath.Dir(log), 0o700); err != nil {
			return ""
		}
		if err := m.sys.FS.WriteFile(log, append(before, line...), 0o600); err != nil {
			return ""
		}
		if home := m.sys.Env.Home(); home != "" && strings.HasPrefix(log, home+"/") {
			return "~" + strings.TrimPrefix(log, home)
		}
		return log
	}
}
