package cli

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Files interface {
	Stat(path string) (fs.FileInfo, error)
}

type Options struct {
	Home string
	// value: the pool whose run asks `gh auth status` once for every reader.
	Pool *lanes.Pool
}

type Inspector struct {
	run   Runner
	files Files
	o     Options
}

var _ contract.Inspector = (*Inspector)(nil)

func New(run Runner, files Files, o Options) *Inspector {
	return &Inspector{run: run, files: files, o: o}
}

type read struct {
	cmd       local.Cmd
	shared    bool
	interpret func(out string) settings.Fact
}

func reads() map[catalog.Fact]read {
	return map[catalog.Fact]read{
		catalog.FactGitEmail:   {cmd: local.Command("git", "config", "--global", "user.email"), interpret: present("")},
		catalog.FactGitName:    {cmd: local.Command("git", "config", "--global", "user.name"), interpret: present("")},
		catalog.FactGitSigning: {cmd: local.Command("git", "config", "--global", "commit.gpgsign"), interpret: signing},
		catalog.FactGPGKey:     {cmd: local.Command("gpg", "--list-secret-keys", "--keyid-format", "long"), interpret: present("~/.gnupg")},
		catalog.FactGitHub:     {cmd: local.Command("gh", "auth", "status"), shared: true, interpret: github},
	}
}

func (i *Inspector) Read(ctx context.Context, f catalog.Fact) (settings.Fact, error) {
	if f == catalog.FactSSHKey {
		return i.sshKey(), nil
	}
	r, ok := reads()[f]
	if !ok {
		return settings.Unread(), nil
	}
	out, err := i.output(ctx, r)
	if errors.Is(err, local.ErrTimeout) || ctx.Err() != nil {
		return settings.Fact{}, local.ErrTimeout
	}
	if fact, unanswered := unanswered(f, out, err); unanswered {
		return fact, nil
	}
	return r.interpret(out.Stdout + "\x00" + out.Stderr), nil
}

func (i *Inspector) sshKey() settings.Fact {
	if _, err := i.files.Stat(path.Join(i.o.Home, ".ssh", "id_ed25519")); err != nil {
		return settings.Fact{State: catalog.StateAbsent}
	}
	return settings.Fact{State: catalog.StateOK, Value: "present", Detail: "~/.ssh/id_ed25519"}
}

// invariant: a tool that is missing or fails is an answer: the fact is
// absent (signing off), never an error.
func unanswered(f catalog.Fact, out local.Output, err error) (settings.Fact, bool) {
	switch {
	case err == nil && out.ExitCode == 0:
		return settings.Fact{}, false
	case f == catalog.FactGitSigning:
		return settings.Fact{State: catalog.StateAbsent, Value: "off"}, true
	}
	return settings.Fact{State: catalog.StateAbsent}, true
}

// invariant: a command another reader asks too (gh auth status, read by
// config) runs once per run of the pool.
func (i *Inspector) output(ctx context.Context, r read) (local.Output, error) {
	if !r.shared || i.o.Pool == nil {
		return i.run.Output(ctx, r.cmd)
	}
	return lanes.Shared(ctx, i.o.Pool, lanes.Command(r.cmd.Name, r.cmd.Args), func(ctx context.Context) (local.Output, error) {
		return i.run.Output(ctx, r.cmd)
	})
}

func stdout(out string) string {
	text, _, _ := strings.Cut(out, "\x00")
	return strings.TrimSpace(text)
}

func present(detail string) func(string) settings.Fact {
	return func(out string) settings.Fact {
		got := stdout(out)
		switch {
		case got == "":
			return settings.Fact{State: catalog.StateAbsent}
		case detail != "":
			return settings.Fact{State: catalog.StateOK, Value: "present", Detail: detail}
		}
		return settings.Fact{State: catalog.StateOK, Value: got}
	}
}

func signing(out string) settings.Fact {
	if stdout(out) == "true" {
		return settings.Fact{State: catalog.StateOK, Value: "on"}
	}
	return settings.Fact{State: catalog.StateAbsent, Value: "off"}
}

// outside: gh prints `Logged in to github.com account <user> (keyring)`, on stderr
// or stdout by version.
func github(out string) settings.Fact {
	_, rest, found := strings.Cut(strings.ReplaceAll(out, "\x00", ""), "account ")
	account, _, _ := strings.Cut(rest, " ")
	if !found || account == "" {
		return settings.Fact{State: catalog.StateOK, Value: "logged in"}
	}
	return settings.Fact{State: catalog.StateOK, Value: account}
}
