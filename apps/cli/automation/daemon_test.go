package automation

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/project"
)

func watchingProject(t *testing.T, path string) (root, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root, outside = filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	for _, d := range []string{root, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := project.Init(root, project.InitOpts{Name: "site"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	manifest := "version: 1\nname: site\nworlds:\n  neo:\n    agents: [neo]\n    automations:\n      watch:\n        agent: neo\n        on:\n          fs:\n            path: " + strconv.Quote(path) + "\n        run: \"say hi\"\n"
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root, outside
}

func runDaemon(t *testing.T, terminal bool, answer string) (string, error) {
	t.Helper()
	was := stdinIsTerminal
	stdinIsTerminal = func() bool { return terminal }
	t.Cleanup(func() { stdinIsTerminal = was })
	var errOut bytes.Buffer
	daemonCmd.SetIn(strings.NewReader(answer))
	daemonCmd.SetErr(&errOut)
	daemonCmd.SetOut(&bytes.Buffer{})
	t.Cleanup(func() { daemonCmd.SetIn(nil); daemonCmd.SetErr(nil); daemonCmd.SetOut(nil) })
	err := daemonCmd.RunE(daemonCmd, nil)
	return errOut.String(), err
}

func TestTheDaemonRefusesAnOutsideWatchOffATerminal(t *testing.T) {
	root, outside := watchingProject(t, "../outside")
	_, err := runDaemon(t, false, "")
	var refused *cliproject.NotGrantedError
	if !errors.As(err, &refused) {
		t.Fatalf("daemon = %v, want a NotGrantedError", err)
	}
	want := "outside.not-granted: spwn.yaml names a host path outside the project " + strconv.Quote(root) + ", and no answer grants it: " + strconv.Quote(outside) +
		` (watched by automation "watch" of world "neo": its files' names and times are recorded in the receipts and passed to the project's agent)` + "\n" + daemonRefusalHint
	if err.Error() != want {
		t.Fatalf("refusal = %q\nwant      %q", err, want)
	}
	if _, serr := os.Lstat(cliproject.GrantsPath()); serr == nil {
		t.Error("a refusal wrote a grant")
	}
}

func TestTheDaemonAsksOnATerminalAndANoRefuses(t *testing.T) {
	root, outside := watchingProject(t, "../outside")
	asked, err := runDaemon(t, true, "n\n")
	if err == nil || !strings.Contains(err.Error(), "were not allowed") {
		t.Fatalf("a no was taken: %v", err)
	}
	want := "\n  spwn.yaml names host paths outside the project " + strconv.Quote(root) + ":\n" +
		"    " + strconv.Quote(outside) + "\n" +
		`      watched by automation "watch" of world "neo": its files' names and times are recorded in the receipts and passed to the project's agent` + "\n" +
		"  The world's agents run what this project gives them, with this access.\n" +
		"  Allow them for this project folder? A yes is kept in " + cliproject.GrantsPath() + ". [y/N] "
	if asked != want {
		t.Fatalf("the ask reads\n%q\nwant\n%q", asked, want)
	}
	if _, serr := os.Lstat(cliproject.GrantsPath()); serr == nil {
		t.Error("a no wrote a grant")
	}
}

func TestADaemonWatchPathIsCheckedBeforeTheAsk(t *testing.T) {
	for path, code := range map[string]string{"~/inbox": cliproject.TildeCode, "../outside\x1b[2K": cliproject.ControlCode} {
		watchingProject(t, path)
		asked, err := runDaemon(t, true, "y\n")
		var refused *cliproject.DeclaredError
		if !errors.As(err, &refused) || refused.Code() != code {
			t.Errorf("%q: %v, want %s", path, err, code)
		}
		if asked != "" {
			t.Errorf("%q was asked: %q", path, asked)
		}
	}
}

func TestAWatchPathSwappedToALinkOutIsRefusedWhenTheWatchStarts(t *testing.T) {
	root, outside := watchingProject(t, "inbox")
	inbox := filepath.Join(root, "inbox")
	if err := os.Mkdir(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := cliproject.Require()
	if err != nil {
		t.Fatal(err)
	}
	reaches, declared, err := watchReaches(p)
	if err != nil || len(reaches) != 0 {
		t.Fatalf("watchReaches = %v, %v", reaches, err)
	}
	want := []cliproject.Declared{{Text: "inbox", Resolved: inbox}}
	if len(declared) != 1 || declared[0] != want[0] {
		t.Fatalf("declared %+v, want %+v", declared, want)
	}
	if err := os.Remove(inbox); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, inbox); err != nil {
		t.Fatal(err)
	}
	var changed *cliproject.ChangedError
	if err := cliproject.Recheck(p.Root, declared); !errors.As(err, &changed) {
		t.Fatalf("Recheck = %v, want a ChangedError", err)
	}
}
