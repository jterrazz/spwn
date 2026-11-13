package keychain_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the login keychain as `security` answers, every command line it was
// given recorded.
type login struct {
	items map[string]string
	argv  []string
	stdin []string
}

// outside: `security -i` reads one command per line, its words split on
// spaces and double-quoted strings, a backslash escaping the next character.
func words(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, escaped, started := false, false, false
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			quoted, started = !quoted, true
		case r == ' ' && !quoted:
			if started {
				out = append(out, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	if started {
		out = append(out, cur.String())
	}
	return out
}

func flag(args []string, name string) string {
	i := slices.Index(args, name)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func (l *login) play(c local.Cmd) (local.Output, error) {
	l.argv = append(l.argv, strings.Join(append([]string{c.Name}, c.Args...), " "))
	args := c.Args
	if len(args) == 1 && args[0] == "-i" {
		l.stdin = append(l.stdin, c.Stdin)
		args = words(strings.TrimSpace(c.Stdin))
	}
	switch {
	case len(args) > 0 && args[0] == "add-generic-password" && flag(args, "-s") == "station":
		l.items[flag(args, "-a")] = flag(args, "-w")
	case len(args) > 0 && args[0] == "find-generic-password":
		v, ok := l.items[flag(args, "-a")]
		if !ok {
			return local.Output{ExitCode: 44, Stderr: "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain."}, nil
		}
		return local.Output{Stdout: v + "\n"}, nil
	case len(args) > 0 && args[0] == "delete-generic-password":
		if _, ok := l.items[flag(args, "-a")]; !ok {
			return local.Output{ExitCode: 44}, nil
		}
		delete(l.items, flag(args, "-a"))
	case len(args) == 1 && args[0] == "dump-keychain":
		var out strings.Builder
		out.WriteString("keychain: \"/Users/owner/Library/Keychains/login.keychain-db\"\nclass: \"genp\"\nattributes:\n    \"acct\"<blob>=\"wifi\"\n    \"svce\"<blob>=\"AirPort\"\n")
		for _, a := range slices.Sorted(maps.Keys(l.items)) {
			out.WriteString("keychain: \"/Users/owner/Library/Keychains/login.keychain-db\"\nversion: 512\nclass: \"genp\"\nattributes:\n    0x00000007 <blob>=\"station\"\n    \"acct\"<blob>=\"" + a + "\"\n    \"svce\"<blob>=\"station\"\n")
		}
		return local.Output{Stdout: out.String()}, nil
	default:
		return local.Output{ExitCode: 97, Stderr: "unexpected: " + strings.Join(args, " ")}, nil
	}
	return local.Output{}, nil
}

func TestVault(t *testing.T) {
	contracttest.Vault(t, func(*testing.T) contract.Vault {
		l := &login{items: map[string]string{}}
		return keychain.New(local.Played(l.play, "security"))
	})
}

func TestASecretTravelsOnStdinNeverInArgv(t *testing.T) {
	const kept = "sk-ant-oat01-SENTINEL"
	l := &login{items: map[string]string{}}
	v := keychain.New(local.Played(l.play, "security"))
	if err := v.Put(t.Context(), "operator:claude:oauth-token", []byte(kept)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	for _, argv := range l.argv {
		if strings.Contains(argv, "SENTINEL") {
			t.Errorf("a command line carries the secret: %s", argv)
		}
	}
	if len(l.stdin) != 1 || !strings.Contains(l.stdin[0], kept) {
		t.Errorf("stdin = %q; want the one add command carrying the secret", l.stdin)
	}
}

func TestAMultilineSecretIsRefusedBeforeSecurityRuns(t *testing.T) {
	l := &login{items: map[string]string{}}
	v := keychain.New(local.Played(l.play, "security"))
	if err := v.Put(t.Context(), "session:a", []byte("one\ntwo")); err == nil {
		t.Fatal("a secret holding a newline was put")
	}
	if len(l.argv) != 0 {
		t.Fatalf("security ran: %v", l.argv)
	}
}
