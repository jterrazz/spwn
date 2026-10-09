package gh

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestLogin_NoGhBinary fails fast with a typed error when gh isn't
// on PATH. The CLI consumes ErrGhNotInstalled to render an
// install-gh hint instead of a generic exec failure.
func TestLogin_NoGhBinary(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("PATH", tmp) // no gh here

	err := Login(context.Background(), nil)
	if !errors.Is(err, ErrGhNotInstalled) {
		t.Fatalf("want ErrGhNotInstalled, got %v", err)
	}
}

// TestLogin_NoHostLogin: gh exists but `gh auth token` fails. We
// surface ErrHostNotLoggedIn so the CLI tells the user to run
// `gh auth login` rather than crashing on exec.
func TestLogin_NoHostLogin(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Stub gh that always exits non-zero — simulates "not logged
	// in" without actually running gh.
	stubPath := tmp + "/gh"
	stub := "#!/bin/bash\nexit 1\n"
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	err := Login(context.Background(), nil)
	if !errors.Is(err, ErrHostNotLoggedIn) {
		t.Fatalf("want ErrHostNotLoggedIn, got %v", err)
	}
}

// TestLogin_HappyPath stubs gh to return a token + user, then
// asserts the resulting hosts.yml has the inline-token shape.
// Doesn't actually hit the network or require a real gh.
func TestLogin_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	stubPath := tmp + "/gh"
	// Stub differentiates between `gh auth token` (echoes token)
	// and `gh api user --jq .login` (echoes username). Anything
	// else exits 0 silently.
	stub := `#!/bin/bash
case "$1 $2" in
  "auth token") echo "gho_stub-token-1234" ;;
  "api user") echo "stubuser" ;;
esac
`
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	if err := Login(context.Background(), nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !IsAuthenticated() {
		t.Fatal("expected authenticated after Login")
	}
	b, err := os.ReadFile(HostsPath())
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"github.com:",
		"oauth_token: gho_stub-token-1234",
		"user: stubuser",
		"git_protocol: https",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hosts.yml missing %q\n--- got ---\n%s", want, got)
		}
	}
}
