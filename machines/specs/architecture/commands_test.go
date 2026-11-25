package architecture_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	command "github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/wire"
)

// value: the command tree station answers with, built by wire.Build as a run
// builds it, over an empty home, and never run; with help, which cobra adds
// to it when a run starts.
func commandTree(t *testing.T) *command.Command {
	t.Helper()
	dir := t.TempDir()
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = null.Close(), out.Close() })
	env := []string{
		"HOME=" + dir, "PATH=" + filepath.Join(dir, "bin"), "TZ=UTC", "STATION_MACHINE_KIND=mac",
		"STATION_SYSTEM_DIRS=", "STATION_SYSTEM_ROOT=" + filepath.Join(dir, "root"),
	}
	p, code := wire.Build(nil, env, null, out, out)
	if p == nil {
		t.Fatalf("station built no tree: exit %d", code)
	}
	t.Cleanup(func() { _ = p.Close() })
	root := p.Root()
	root.InitDefaultHelpCmd()
	return root
}

// value: every command a user types, depth first, as help shows them: never
// the root, whose bare run is its overview, nor a hidden command or one under
// it.
func typedCommands(root *command.Command) []*command.Command {
	var out []*command.Command
	var walk func(c *command.Command)
	walk = func(c *command.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden {
				continue
			}
			out = append(out, sub)
			walk(sub)
		}
	}
	walk(root)
	return out
}

func commandPath(c *command.Command) string {
	return strings.TrimSpace(strings.TrimPrefix(c.CommandPath(), c.Root().Name()))
}

func top(c *command.Command) bool { return c.HasParent() && !c.Parent().HasParent() }

// value: a breach today's tree still holds, written down with the lot of the
// plan that removes it and the day it was; one whose breach is gone fails
// until its line is deleted.
type exception struct{ rule, where, lot, since string }

func excuse(t *testing.T, found []finding, excepted []exception) {
	t.Helper()
	slices.SortFunc(found, func(a, b finding) int { return strings.Compare(a.where+a.what, b.where+b.what) })
	found = slices.Compact(found)
	for _, f := range found {
		if !slices.ContainsFunc(excepted, func(e exception) bool { return e.rule == f.rule && e.where == f.where }) {
			t.Errorf("%s: %s", f.where, f.what)
		}
	}
	for _, e := range excepted {
		if !slices.ContainsFunc(found, func(f finding) bool { return e.rule == f.rule && e.where == f.where }) {
			t.Errorf("%s: the %s exception %s owns (written %s) holds no longer: delete its line", e.where, e.rule, e.lot, e.since)
		}
	}
}

func bites(t *testing.T, rule string, found []finding, where string) {
	t.Helper()
	if !slices.ContainsFunc(found, func(f finding) bool { return f.rule == rule && f.where == where }) {
		t.Errorf("the %s rule did not report %q: %v", rule, where, found)
	}
}

// value: the seed world a run of the spec build is judged over: the one
// whose machines, apps and settings the help's examples name.
const exampleWorld = "stories"

var (
	specBuildOnce sync.Once
	specBuild     string
	specBuildErr  error
)

// value: the spec build of station, built once for every rule that runs it,
// in a folder removed when the package's tests end.
func specBinary(t *testing.T) string {
	t.Helper()
	specBuildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "station-tree-")
		if err != nil {
			specBuildErr = err
			return
		}
		specBuild = filepath.Join(dir, "station-test")
		build := exec.Command("go", "build", "-tags", "world", "-o", specBuild, "./apps/engine/cmd/station")
		build.Dir = repoRoot(t)
		if out, err := build.CombinedOutput(); err != nil {
			specBuildErr = errors.New(err.Error() + "\n" + string(out))
		}
	})
	if specBuildErr != nil {
		t.Fatalf("go build -tags world: %v", specBuildErr)
	}
	return specBuild
}

func TestMain(m *testing.M) {
	code := m.Run()
	if specBuild != "" {
		_ = os.RemoveAll(filepath.Dir(specBuild))
	}
	os.Exit(code)
}

type ran struct {
	words          []string
	exit           int
	stdout, stderr string
	timedOut       bool
}

// value: one run of the spec build over a fresh copy of the example world,
// with no terminal: what a script that typed those words would meet.
func runOverWorld(t *testing.T, bin string, words []string) ran {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"bin", "root"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	args := make([]string, len(words))
	for i, w := range words {
		if rest, ok := strings.CutPrefix(w, "~"); ok {
			w = dir + rest
		}
		args[i] = w
	}
	env := []string{
		"HOME=" + dir, "PATH=" + filepath.Join(dir, "bin"), "NO_COLOR=1", "COLUMNS=80", "TZ=UTC",
		"STATION_GITHUB_API_URL=http://127.0.0.1:9", "STATION_MACHINE_KIND=mac",
		"STATION_KERNEL_STUB=" + filepath.Join(dir, "kernel"), "STATION_POWER_STUB=" + filepath.Join(dir, "power"),
		"STATION_SYSTEM_DIRS=", "STATION_SYSTEM_ROOT=" + filepath.Join(dir, "root"),
		"STATION_WORLD=" + filepath.Join(dir, "world.json"), "STATION_TERMINAL=spec-terminal",
		"STATION_WORLD_SEED=" + filepath.Join(repoRoot(t), "specs", "usecases", "worlds", exampleWorld+".json"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir, cmd.Env = dir, env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := ran{words: words, stdout: stdout.String(), stderr: stderr.String(), timedOut: ctx.Err() != nil}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		r.exit = exit.ExitCode()
	case err != nil:
		r.exit = -1
	}
	return r
}
