package macos

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

var restartLine = catalog.AsRoot("fdesetup authrestart -delayminutes 0")

const (
	restartBudget = 30 * time.Second
	// outside: ssh exits 255 when the Mac drops the connection.
	dropped = 255
)

type Dirs interface {
	MkdirAll(path string, perm fs.FileMode) error
}

type unlocker struct {
	run        local.Runner
	dirs       Dirs
	knownHosts string
}

func New(run local.Runner, dirs Dirs, knownHosts string) contract.Unlocker {
	return &unlocker{run: run, dirs: dirs, knownHosts: knownHosts}
}

func batch() []string { return []string{"-o", "ConnectTimeout=2", "-o", "BatchMode=yes"} }

func (u *unlocker) Unlock(ctx context.Context, mac, lan string) error {
	if err := u.dirs.MkdirAll(u.knownHosts, 0o700); err != nil {
		return err
	}
	// invariant: pre-boot sshd advertises another host key than the running OS, so
	// the whole conversation keeps its own known_hosts file.
	preboot := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + path.Join(u.knownHosts, mac+".preboot"),
	}
	// outside: an option given to ssh wins over its block, so HostName dials
	// the LAN address while the block for mac still names its login and key.
	if lan != "" {
		preboot = append(preboot, "-o", "HostName="+lan)
	}
	out, err := u.run.Output(ctx, local.Cmd{Name: "ssh", Args: concat(batch(), preboot, []string{mac, "true"})})
	if err != nil {
		return contract.ErrUnreachable
	}
	// outside: a Mac in FileVault's pre-boot state answers but refuses the key with
	// "Permission denied": that is the state unlock exists for, not unreachable.
	if out.ExitCode != 0 && !strings.Contains(out.Stderr, "Permission denied") {
		return contract.ErrUnreachable
	}
	// outside: ssh disconnects as soon as the password unlocks the disk, so the
	// session's own exit is the success path and carries no verdict.
	_, err = u.run.Attach(ctx, local.Cmd{Name: "ssh", Args: concat(
		[]string{"-o", "PreferredAuthentications=password", "-o", "PubkeyAuthentication=no"}, preboot, []string{mac},
	)}, local.ToStdout)
	return err
}

func (u *unlocker) PlannedRestart(ctx context.Context, mac string) error {
	out, err := u.run.Output(ctx, local.Cmd{
		Name: "ssh", Args: concat(batch(), []string{mac, restartLine}),
		Timeout: restartBudget,
	})
	if err != nil && !errors.Is(err, local.ErrTimeout) {
		return err
	}
	if err == nil && out.ExitCode != 0 && out.ExitCode != dropped {
		return &contract.RestartRefusedError{Detail: firstLine(out.Stderr + out.Stdout)}
	}
	return nil
}

func concat(parts ...[]string) []string {
	var out []string
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return "no detail"
}
