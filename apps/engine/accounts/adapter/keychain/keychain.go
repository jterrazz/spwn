package keychain

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const (
	command = "security"
	entry   = "station"
	// outside: `security` exits 44 when the item is not in the keychain.
	notFound = 44
)

var errMultiline = errors.New("a keychain value holds no newline")

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Keychain struct{ run Runner }

var _ contract.Vault = Keychain{}

func New(run Runner) Keychain { return Keychain{run: run} }

func (k Keychain) security(ctx context.Context, stdin string, args ...string) (local.Output, error) {
	return k.run.Output(ctx, local.Cmd{Name: command, Args: args, Stdin: stdin})
}

func failed(name string, out local.Output, err error) error {
	if err == nil {
		err = errors.New(strings.TrimSpace(out.Stderr))
	}
	return &accounts.VaultError{Name: name, Cause: err}
}

func (k Keychain) Get(ctx context.Context, name string) ([]byte, error) {
	out, err := k.security(ctx, "", "find-generic-password", "-s", entry, "-a", name, "-w")
	switch {
	case err != nil:
		return nil, failed(name, out, err)
	case out.ExitCode == notFound:
		return nil, accounts.ErrNotKept
	case out.ExitCode != 0:
		return nil, failed(name, out, nil)
	}
	return []byte(strings.TrimRight(out.Stdout, "\n")), nil
}

func (k Keychain) Put(ctx context.Context, name string, secret []byte) error {
	// invariant: the secret travels on security's stdin, in its interactive mode,
	// never in argv (where `ps` shows it) and never through a prompt.
	if strings.ContainsAny(name+string(secret), "\n\r") {
		return &accounts.VaultError{Name: name, Cause: errMultiline}
	}
	line := "add-generic-password -U -s " + entry + " -a " + quote(name) + " -w " + quote(string(secret)) + "\n"
	out, err := k.security(ctx, line, "-i")
	if err != nil || out.ExitCode != 0 {
		return failed(name, out, err)
	}
	return nil
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func (k Keychain) Forget(ctx context.Context, name string) error {
	out, err := k.security(ctx, "", "delete-generic-password", "-s", entry, "-a", name)
	switch {
	case err != nil:
		return failed(name, out, err)
	case out.ExitCode == notFound:
		return accounts.ErrNotKept
	case out.ExitCode != 0:
		return failed(name, out, nil)
	}
	return nil
}

func (k Keychain) Names(ctx context.Context) ([]string, error) {
	// outside: `security dump-keychain` prints one block per item, opened by its
	// `keychain:` line, its attributes as `"acct"<blob>="name"`.
	out, err := k.security(ctx, "", "dump-keychain")
	if err != nil || out.ExitCode != 0 {
		return nil, failed("", out, err)
	}
	var names []string
	var account, svc string
	flush := func() {
		if svc == entry && account != "" && !slices.Contains(names, account) {
			names = append(names, account)
		}
		account, svc = "", ""
	}
	for _, line := range strings.Split(out.Stdout, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "keychain:"):
			flush()
		case strings.HasPrefix(line, `"acct"<blob>=`):
			account = unquote(strings.TrimPrefix(line, `"acct"<blob>=`))
		case strings.HasPrefix(line, `"svce"<blob>=`):
			svc = unquote(strings.TrimPrefix(line, `"svce"<blob>=`))
		}
	}
	flush()
	slices.Sort(names)
	return names, nil
}

func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return ""
}
