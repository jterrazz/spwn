package world

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
)

const askHeading = "spwn.yaml names host paths outside the project"

type composeRun struct {
	root, outsideB, linkText, grants string
	terminal                         *os.File
	started                          []string
}

func twoWorldProject(t *testing.T, failing string) *composeRun {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	r := &composeRun{root: filepath.Join(dir, "site"), outsideB: filepath.Join(dir, "outside-b"),
		linkText: "/nowhere-\x1b[2J\x1b[1;5r\u202e/kb", grants: cliproject.GrantsPath()}
	for _, d := range []string{r.root, r.outsideB} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := project.Init(r.root, project.InitOpts{Name: "site"}); err != nil {
		t.Fatal(err)
	}
	manifest := "version: 1\nname: site\nworlds:\n  a:\n    knowledge: kn\n  b:\n    workspaces: [" + strconv.Quote(r.outsideB) + "]\n"
	if err := os.WriteFile(filepath.Join(r.root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(r.linkText, filepath.Join(r.root, "kn")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(r.root)

	f, err := os.Create(filepath.Join(dir, "terminal"))
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = f
	saved := []any{startWorld, askTerminal, runningWorld}
	t.Cleanup(func() {
		os.Stderr = stderr
		startWorld = saved[0].(func(*cobra.Command, []string, *plannedWorld) error)
		askTerminal = saved[1].(func() bool)
		runningWorld = saved[2].(func(context.Context, string) *world.World)
		f.Close()
	})
	r.terminal = f
	askTerminal = func() bool { return true }
	runningWorld = func(context.Context, string) *world.World { return nil }
	startWorld = func(_ *cobra.Command, args []string, _ *plannedWorld) error {
		name := args[0]
		if !strings.Contains(r.printed(t), askHeading) {
			t.Errorf("world %s started before the ask", name)
		}
		if name == "b" && strings.Contains(r.kept(t), r.outsideB) {
			t.Errorf("b's path was kept before world b started")
		}
		r.started = append(r.started, name)
		ui.New().Warn("Warning", "skill file tools/x/skills/\x1b[2J\x1b[1;5r\u202e.md is a link out of the tool folder: not mounted")
		if name == failing {
			return errors.New("spawn failed")
		}
		return nil
	}
	return r
}

func (r *composeRun) printed(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(r.terminal.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (r *composeRun) up(t *testing.T, answer string) error {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(answer))
	cmd.SetErr(r.terminal)
	return composeUpRunE(cmd, nil)
}

func (r *composeRun) kept(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(r.grants)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestATwoWorldUpAsksOnceBeforeAnyWorldStartsAndPrintsNoProjectByteRaw(t *testing.T) {
	r := twoWorldProject(t, "")
	if err := r.up(t, "y\n"); err != nil {
		t.Fatalf("spwn up: %v", err)
	}
	out := r.printed(t)
	if strings.ContainsAny(out, "\x1b\u202e") {
		t.Errorf("a project byte reached the terminal raw:\n%q", out)
	}
	if n := strings.Count(out, askHeading); n != 1 {
		t.Fatalf("asked %d times, want once:\n%s", n, out)
	}
	ask := out[strings.Index(out, askHeading):]
	for _, path := range []string{r.outsideB, r.linkText} {
		if !strings.Contains(ask, strconv.Quote(path)) {
			t.Errorf("the ask does not list %q:\n%s", path, ask)
		}
	}
	if strings.Join(r.started, ",") != "a,b" {
		t.Errorf("started %v, want a then b", r.started)
	}
	kept := r.kept(t)
	if !strings.Contains(kept, r.outsideB) || !strings.Contains(kept, "nowhere-") {
		t.Errorf("each world's yes was not kept once it started:\n%s", kept)
	}
}

func TestANoToTheOneAskStartsNoWorldAndKeepsNothing(t *testing.T) {
	r := twoWorldProject(t, "")
	if err := r.up(t, "n\n"); err == nil {
		t.Fatal("a no was taken for a yes")
	}
	if len(r.started) != 0 {
		t.Errorf("a no started %v", r.started)
	}
	if kept := r.kept(t); kept != "" {
		t.Errorf("a no kept:\n%s", kept)
	}
}

func TestAWorldThatFailsToStartKeepsNoneOfItsPaths(t *testing.T) {
	r := twoWorldProject(t, "b")
	if err := r.up(t, "y\n"); err == nil {
		t.Fatal("the failed start of b was not reported")
	}
	kept := r.kept(t)
	if strings.Contains(kept, r.outsideB) {
		t.Errorf("b failed to start and its path was kept:\n%s", kept)
	}
	if !strings.Contains(kept, "nowhere-") {
		t.Errorf("a started and its path was not kept:\n%s", kept)
	}
}
