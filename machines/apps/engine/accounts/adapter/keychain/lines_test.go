package keychain_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: the lines a grant runs in a VM, run here by sh in a home of
// the test's own: the token replaces the program's one line and keeps the
// others, in a file of mode 0600, and the removal takes that line alone.
func TestTheStoreLinesReplaceTheTokenAloneInAFileOfMode0600(t *testing.T) {
	home := t.TempDir()
	sys := local.New([]string{"HOME=" + home, "PATH=/usr/bin:/bin"}, nil, nil, nil)
	rc := filepath.Join(home, ".npmrc")
	if err := sys.FS.WriteFile(rc, []byte("registry=https://registry.example/\n//registry.npmjs.org/:_authToken=OLD000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := bare()
	sh := func(line, stdin string) {
		t.Helper()
		out, err := sys.Run.Output(t.Context(), local.Cmd{Name: "sh", Args: []string{"-c", line}, Stdin: stdin, Env: []string{"HOME=" + home, "PATH=/usr/bin:/bin"}})
		if err != nil || out.ExitCode != 0 {
			t.Fatalf("sh %q = %+v, %v", line, out, err)
		}
	}
	sh(d.Inject, "FAKE000\n")
	got, _ := sys.FS.ReadFile(rc)
	if string(got) != "registry=https://registry.example/\n//registry.npmjs.org/:_authToken=FAKE000\n" {
		t.Fatalf(".npmrc = %q", got)
	}
	if info, err := sys.FS.Stat(rc); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf(".npmrc mode = %v, %v; want 0600", info.Mode().Perm(), err)
	}
	sh(d.Remove, "")
	if got, _ := sys.FS.ReadFile(rc); string(got) != "registry=https://registry.example/\n" || strings.Contains(string(got), "FAKE000") {
		t.Fatalf(".npmrc after removal = %q", got)
	}
}
