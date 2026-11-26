package keychain_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the login keychain as `security` answers it, by item name and
// account; locked fails every command as a keychain that cannot be read.
type login struct {
	items  map[string]string
	locked bool
	argv   [][]string
}

func flag(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

// outside: `security -i` reads one command per line, a quoted word kept
// whole; the head holds no quote nor backslash.
func words(line string) []string {
	var out []string
	for i, part := range strings.Split(strings.TrimSpace(line), "\"") {
		if i%2 == 1 {
			out = append(out, part)
			continue
		}
		out = append(out, strings.Fields(part)...)
	}
	return out
}

func (l *login) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	l.argv = append(l.argv, append([]string{c.Name}, c.Args...))
	if c.Name != "/usr/bin/security" {
		return local.Output{}, errors.New("not security: " + c.Name)
	}
	if l.locked {
		return local.Output{ExitCode: 36, Stderr: "security: SecKeychainSearchCopyNext: User interaction is not allowed."}, nil
	}
	args := c.Args
	if len(args) == 1 && args[0] == "-i" {
		args = words(c.Stdin)
	}
	switch {
	case len(args) > 0 && args[0] == "add-generic-password":
		l.items[flag(args, "-s")+"/"+flag(args, "-a")] = flag(args, "-w")
		return local.Output{}, nil
	case len(args) > 0 && args[0] == "find-generic-password":
		v, ok := l.items[flag(args, "-s")+"/"+flag(args, "-a")]
		if !ok {
			return local.Output{ExitCode: 44}, nil
		}
		return local.Output{Stdout: v + "\n"}, nil
	}
	return local.Output{ExitCode: 1}, nil
}

func TestAnchor(t *testing.T) {
	contracttest.Anchor(t, func(*testing.T) contract.Anchor { return keychain.New(&login{items: map[string]string{}}, "") })
}

func TestTheHeadIsAnItemOfItsOwnNeverAVaultSecret(t *testing.T) {
	l := &login{items: map[string]string{}}
	h := devices.Book{Generation: 2}.Head()
	if err := keychain.New(l, "").Write(t.Context(), h); err != nil {
		t.Fatal(err)
	}
	if got := l.items["station-devices/head"]; got != h.String() {
		t.Fatalf("items %v; want the head under station-devices", l.items)
	}
	for _, argv := range l.argv {
		if strings.Contains(strings.Join(argv, " "), h.Digest) {
			t.Fatalf("the head rode argv: %v", argv)
		}
	}
}

func TestALockedKeychainIsAnErrorNeverNoHead(t *testing.T) {
	var unreadable *contract.AnchorUnreadableError
	if _, err := keychain.New(&login{locked: true}, "").Read(t.Context()); !errors.As(err, &unreadable) {
		t.Fatalf("Read = %v; want the keychain unreadable", err)
	}
	l := &login{items: map[string]string{"station-devices/head": "not a head"}}
	if _, err := keychain.New(l, "").Read(t.Context()); !errors.As(err, &unreadable) {
		t.Fatalf("Read of a damaged head = %v; want it unreadable", err)
	}
}
