package cliproject

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func projects(t *testing.T) (a, b, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	a, b, outside = filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "outside")
	for _, d := range []string{a, b, outside, filepath.Join(a, "inside")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return a, b, outside
}

func noGrantsFile(t *testing.T) {
	t.Helper()
	if _, err := os.Lstat(GrantsPath()); err == nil {
		t.Error("the grants file was written")
	}
}

func TestResolveTellsWhatLeavesTheProject(t *testing.T) {
	a, _, outside := projects(t)
	if err := os.Symlink(outside, filepath.Join(a, "away")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "gone"), filepath.Join(a, "dangling")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("inside", filepath.Join(a, "in")); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{
		".": false, "inside": false, "./inside/../inside": false, "missing": false, "~/x": false, "in/x": false,
		"..": true, "../outside": true, outside: true, "/": true, "away": true, "away/sub": true, "dangling": true,
	} {
		if _, got := Resolve(a, p); got != want {
			t.Errorf("Resolve(%q) leaves = %v, want %v", p, got, want)
		}
	}
	for p, want := range map[string]string{
		"away":     outside,
		"away/sub": filepath.Join(outside, "sub"),
		"dangling": filepath.Join(outside, "gone"),
		"in/x":     filepath.Join(a, "inside", "x"),
	} {
		if got, _ := Resolve(a, p); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", p, got, want)
		}
	}
}

func TestResolveReadsNothingOutsideTheProject(t *testing.T) {
	a, _, outside := projects(t)
	target := filepath.Join(outside, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(outside, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(a, "away")); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{
		filepath.Join(outside, "link", "x"): filepath.Join(outside, "link", "x"),
		"away/link":                         filepath.Join(outside, "link"),
		"../outside/link":                   filepath.Join(outside, "link"),
	} {
		if got, leaves := Resolve(a, p); got != want || !leaves {
			t.Errorf("Resolve(%q) = %q %v, want %q unread: an outside link was followed", p, got, leaves, want)
		}
	}
}

func TestCheckRefusesControlCharactersAndATilde(t *testing.T) {
	for _, s := range []string{"x\x1b[1A\x1b[2K", "a\nb", "a\rb", "a\tb", "a\x7fb", "a\u009bb", "a\xffb"} {
		err := CheckText("workspace", s)
		var refused *DeclaredError
		if !errors.As(err, &refused) || refused.Code() != ControlCode {
			t.Errorf("CheckText(%q) = %v, want %s", s, err, ControlCode)
			continue
		}
		if strings.ContainsAny(err.Error(), "\x1b\n\r\t\x7f") {
			t.Errorf("the refusal prints the text raw: %q", err)
		}
	}
	if err := CheckText("workspace", "lib=../lib:ro"); err != nil {
		t.Errorf("a plain entry was refused: %v", err)
	}
	for _, p := range []string{"~", "~/code", " ~user/x"} {
		err := CheckHostPath("workspace path", p)
		var refused *DeclaredError
		if !errors.As(err, &refused) || refused.Code() != TildeCode {
			t.Errorf("CheckHostPath(%q) = %v, want %s", p, err, TildeCode)
		}
	}
	const want = `spwn-yaml.tilde: workspace path "~/code" starts with ~, which spwn.yaml never expands: write the absolute path (one outside the project is asked for once)`
	if err := CheckHostPath("workspace path", "~/code"); err.Error() != want {
		t.Errorf("refusal = %q, want %q", err, want)
	}
	if err := CheckHostPath("workspace path", "a/~b"); err != nil {
		t.Errorf("a ~ inside a path was refused: %v", err)
	}
}

func TestAColonInAHostPathIsRefusedDeclaredOrResolved(t *testing.T) {
	a, _, _ := projects(t)
	for _, p := range []string{"a:", "x:/workspaces/n", "../outside/:ro"} {
		var refused *DeclaredError
		if _, _, err := ResolveHostPath(a, "workspace path", p); !errors.As(err, &refused) || refused.Code() != ColonCode {
			t.Errorf("ResolveHostPath(%q) = %v, want %s", p, err, ColonCode)
		}
	}
	const want = "spwn-yaml.colon: knowledge path \"a:\" holds a colon, which a mount's source cannot: only `name=path:ro` ends a workspace with one"
	if err := CheckHostPath("knowledge path", "a:"); err == nil || err.Error() != want {
		t.Errorf("refusal = %v, want %q", err, want)
	}

	if err := os.Symlink("inside/b:c", filepath.Join(a, "kn")); err != nil {
		t.Fatal(err)
	}
	_, _, err := ResolveHostPath(a, "knowledge path", "kn")
	var refused *DeclaredError
	if !errors.As(err, &refused) || refused.Code() != ColonCode || !strings.Contains(err.Error(), `knowledge path "kn" resolved `) {
		t.Fatalf("a link whose text holds a colon: %v, want %s naming kn", err, ColonCode)
	}
	if resolved, leaves, err := ResolveHostPath(a, "knowledge path", "inside"); err != nil || leaves || resolved != filepath.Join(a, "inside") {
		t.Errorf("a plain inside path: %q %v %v", resolved, leaves, err)
	}
}

func TestConsentOffATerminalRefusesNamingThePathAndTheWay(t *testing.T) {
	a, _, outside := projects(t)
	reach := Reach{Path: outside, Access: "read-write", Grants: `read-write mount at /workspaces/home in world "neo"`}
	_, err := Consent(a, []Reach{reach}, false, false, strings.NewReader(""), &bytes.Buffer{})
	var refused *NotGrantedError
	if !errors.As(err, &refused) || refused.Code() != NotGrantedCode {
		t.Fatalf("Consent = %v, want a NotGrantedError", err)
	}
	want := "outside.not-granted: spwn.yaml names a host path outside the project " + strconv.Quote(a) + ", and no answer grants it: " + strconv.Quote(outside) + ` (read-write mount at /workspaces/home in world "neo")`
	if err.Error() != want {
		t.Fatalf("refusal = %q\nwant      %q", err, want)
	}
	noGrantsFile(t)
}

func TestYesAllowsForThisRunOnlyAndSaysSo(t *testing.T) {
	a, _, outside := projects(t)
	rw := Reach{Path: outside, Access: "read-write", Grants: `read-write mount at /workspaces/home in world "neo"`}
	var out bytes.Buffer
	allowed, err := Consent(a, []Reach{rw}, true, false, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("--yes: %v", err)
	}
	if want := "  allowed for this run: " + strconv.Quote(outside) + ` (read-write mount at /workspaces/home in world "neo")` + "\n"; out.String() != want {
		t.Errorf("--yes prints %q, want %q", out.String(), want)
	}
	if err := allowed.Keep([]Reach{rw}); err != nil {
		t.Fatal(err)
	}
	noGrantsFile(t)
	var refused *NotGrantedError
	if _, err := Consent(a, []Reach{rw}, false, false, strings.NewReader(""), &bytes.Buffer{}); !errors.As(err, &refused) {
		t.Errorf("a --yes served the next run: %v", err)
	}
}

func TestATypedYesIsKeptOnlyOnceTheRunSucceeded(t *testing.T) {
	a, _, outside := projects(t)
	rw := Reach{Path: outside, Access: "read-write", Grants: "read-write mount"}
	allowed, err := Consent(a, []Reach{rw}, false, true, strings.NewReader("y\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("a yes was refused: %v", err)
	}
	noGrantsFile(t)
	var refused *NotGrantedError
	if _, err := Consent(a, []Reach{rw}, false, false, strings.NewReader(""), &bytes.Buffer{}); !errors.As(err, &refused) {
		t.Fatalf("a yes whose run never succeeded was kept: %v", err)
	}
	if err := allowed.Keep([]Reach{rw}); err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= 2; run++ {
		var out bytes.Buffer
		if _, err := Consent(a, []Reach{rw}, false, false, strings.NewReader(""), &out); err != nil || out.Len() != 0 {
			t.Fatalf("run %d after the kept yes: %q %v", run, out.String(), err)
		}
	}
}

func TestAKeptYesServesItsProjectPathAndAccessOnly(t *testing.T) {
	a, b, outside := projects(t)
	ro := Reach{Path: outside, Access: "read-only", Grants: "read-only mount"}
	allowed, err := Consent(a, []Reach{ro}, false, true, strings.NewReader("yes\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := allowed.Keep([]Reach{ro}); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		root  string
		reach Reach
	}{
		"another project":             {b, ro},
		"another path":                {a, Reach{Path: filepath.Dir(outside), Access: "read-only", Grants: "read-only mount"}},
		"read-write on the same path": {a, Reach{Path: outside, Access: "read-write", Grants: "read-write mount"}},
		"a watch on the same path":    {a, Reach{Path: outside, Access: "names and times recorded", Grants: "watched"}},
	} {
		var refused *NotGrantedError
		if _, err := Consent(c.root, []Reach{c.reach}, false, false, strings.NewReader(""), &bytes.Buffer{}); !errors.As(err, &refused) {
			t.Errorf("%s was served by a read-only yes of project a: %v", name, err)
		}
	}
	if !strings.HasPrefix(GrantsPath(), os.Getenv("SPWN_HOME")) {
		t.Fatalf("the grants file %s is not in the user's spwn home", GrantsPath())
	}
}

func TestOnATerminalThePathsAreShownQuotedOnceAndAsked(t *testing.T) {
	a, _, outside := projects(t)
	reaches := []Reach{
		{Path: outside, Access: "read-write", Grants: `read-write mount at /workspaces/home in world "neo"`},
		{Path: filepath.Join(outside, "k\x1b[2K"), Access: "read-write", Grants: `read-write mount at /world/knowledge in world "neo"`},
	}
	var out bytes.Buffer
	if _, err := Consent(a, reaches, false, true, strings.NewReader("n\n"), &out); err == nil {
		t.Fatal("a no was taken as a yes")
	}
	want := "\n  spwn.yaml names host paths outside the project " + strconv.Quote(a) + ":\n" +
		"    " + strconv.Quote(outside) + "\n      read-write mount at /workspaces/home in world \"neo\"\n" +
		"    " + strconv.Quote(filepath.Join(outside, "k\x1b[2K")) + "\n      read-write mount at /world/knowledge in world \"neo\"\n" +
		"  The world's agents run what this project gives them, with this access.\n" +
		"  Allow them for this project folder? A yes is kept in " + GrantsPath() + ". [y/N] "
	if out.String() != want {
		t.Fatalf("the ask reads\n%q\nwant\n%q", out.String(), want)
	}
	if strings.ContainsRune(out.String(), 0x1b) {
		t.Error("the ask prints an escape raw")
	}
	noGrantsFile(t)
}
