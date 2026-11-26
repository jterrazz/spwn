package cliproject

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// projects makes two project folders and an outside folder in a fresh
// temp dir, with the user's spwn home pointed at a fourth.
func projects(t *testing.T) (a, b, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	a, b, outside = filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "outside")
	for _, d := range []string{a, b, outside, filepath.Join(a, "inside")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return a, b, outside
}

func TestResolveTellsWhatLeavesTheProject(t *testing.T) {
	a, _, outside := projects(t)
	if err := os.Symlink(outside, filepath.Join(a, "away")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "gone"), filepath.Join(a, "dangling")); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{
		".": false, "inside": false, "./inside/../inside": false, "missing": false, "~/x": false,
		"..": true, "../outside": true, outside: true, "/": true, "away": true, "away/sub": true, "dangling": true,
	} {
		if _, got := Resolve(a, p); got != want {
			t.Errorf("Resolve(%q) leaves = %v, want %v", p, got, want)
		}
	}
	if got, _ := Resolve(a, "away"); got != outside {
		t.Errorf("a link is shown as %q, want its target %q", got, outside)
	}
}

func TestConsentOffATerminalRefusesNamingThePathAndTheWay(t *testing.T) {
	a, _, outside := projects(t)
	reach := Reach{Path: outside, Access: "read-write", Grants: `read-write mount at /workspaces/home in world "neo"`}
	err := Consent(a, []Reach{reach}, false, false, strings.NewReader(""), &bytes.Buffer{})
	var refused *NotGrantedError
	if !errors.As(err, &refused) || refused.Code() != NotGrantedCode {
		t.Fatalf("Consent = %v, want a NotGrantedError", err)
	}
	want := "outside.not-granted: spwn.yaml names a host path outside the project " + a + ", and no answer grants it: " + outside + ` (read-write mount at /workspaces/home in world "neo")`
	if err.Error() != want {
		t.Fatalf("refusal = %q\nwant      %q", err, want)
	}
	if _, serr := os.Stat(grantsPath()); serr == nil {
		t.Error("a refusal wrote a grant")
	}
}

func TestAYesIsRememberedForItsProjectPathAndAccessOnly(t *testing.T) {
	a, b, outside := projects(t)
	rw := Reach{Path: outside, Access: "read-write", Grants: "read-write mount"}
	if err := Consent(a, []Reach{rw}, true, false, strings.NewReader(""), &bytes.Buffer{}); err != nil {
		t.Fatalf("--yes: %v", err)
	}
	if err := Consent(a, []Reach{rw}, false, false, strings.NewReader(""), &bytes.Buffer{}); err != nil {
		t.Fatalf("the remembered yes was not used: %v", err)
	}
	for name, c := range map[string]struct {
		root  string
		reach Reach
	}{
		"another project": {b, rw},
		"another path":    {a, Reach{Path: filepath.Dir(outside), Access: "read-write", Grants: "read-write mount"}},
		"another access":  {a, Reach{Path: outside, Access: "names and times listed", Grants: "watched"}},
	} {
		var refused *NotGrantedError
		if err := Consent(c.root, []Reach{c.reach}, false, false, strings.NewReader(""), &bytes.Buffer{}); !errors.As(err, &refused) {
			t.Errorf("%s was served by project a's yes: %v", name, err)
		}
	}
	data, err := os.ReadFile(grantsPath())
	if err != nil || !strings.HasPrefix(grantsPath(), os.Getenv("SPWN_HOME")) {
		t.Fatalf("the grant is not in the user's spwn home: %v", err)
	}
	if strings.Contains(string(data), "names and times") {
		t.Errorf("a refusal was remembered: %s", data)
	}
}

func TestOnATerminalThePathsAreShownOnceAndAsked(t *testing.T) {
	a, _, outside := projects(t)
	reaches := []Reach{
		{Path: outside, Access: "read-write", Grants: `read-write mount at /workspaces/home in world "neo"`},
		{Path: filepath.Join(outside, "k"), Access: "read-write", Grants: `read-write mount at /world/knowledge in world "neo"`},
	}
	var out bytes.Buffer
	if err := Consent(a, reaches, false, true, strings.NewReader("n\n"), &out); err == nil {
		t.Fatal("a no was taken as a yes")
	}
	want := "\n  spwn.yaml names host paths outside the project " + a + ":\n" +
		"    " + outside + "\n      read-write mount at /workspaces/home in world \"neo\"\n" +
		"    " + filepath.Join(outside, "k") + "\n      read-write mount at /world/knowledge in world \"neo\"\n" +
		"  The world's agents run what this project gives them, with this access.\n" +
		"  Allow them for this project? The answer is kept in " + grantsPath() + ". [y/N] "
	if out.String() != want {
		t.Fatalf("the ask reads\n%q\nwant\n%q", out.String(), want)
	}
	out.Reset()
	if err := Consent(a, reaches, false, true, strings.NewReader("y\n"), &out); err != nil {
		t.Fatalf("a yes was refused: %v", err)
	}
	out.Reset()
	if err := Consent(a, reaches, false, true, strings.NewReader(""), &out); err != nil || out.Len() != 0 {
		t.Fatalf("a granted path was asked again: %q %v", out.String(), err)
	}
}
