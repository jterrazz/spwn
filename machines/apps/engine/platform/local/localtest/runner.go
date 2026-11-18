package localtest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Runner struct {
	mu     sync.Mutex
	t      testing.TB
	script []*Call
	Paths  map[string]string
	// value: the environment station was started with; its PATH, when it
	// sets one, is what LookPath walks, over Programs and Paths.
	Env []string
	// value: the programs on disk, by absolute path, that a lookup on a PATH
	// finds in the first of its folders that holds one of the name.
	Programs []string
	live     map[int]bool
	Calls    []local.Cmd
	Signals  []local.Sig
}

type Call struct {
	name   string
	args   []string
	output local.Output
	err    error
	answer func() local.Output
	Seen   int
}

func NewRunner(t testing.TB) *Runner {
	return &Runner{t: t, Paths: map[string]string{}, live: map[int]bool{}}
}

func (r *Runner) On(name string, args ...string) *Call {
	c := &Call{name: name, args: args}
	r.script = append(r.script, c)
	return c
}

func (c *Call) Reply(o local.Output) *Call {
	c.output = o
	return c
}

func (c *Call) Answer(f func() local.Output) *Call {
	// value: f answers each call in turn, for a tool whose answer follows
	// what was done to it before.
	c.answer = f
	return c
}

func (c *Call) Fail(err error) *Call {
	c.err = err
	return c
}

func (r *Runner) answer(c local.Cmd) *Call {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls = append(r.Calls, c)
	for _, call := range r.script {
		if call.name == c.Name && slices.Equal(call.args, c.Args) {
			call.Seen++
			if call.answer != nil {
				return &Call{output: call.answer(), err: call.err}
			}
			return call
		}
	}
	r.t.Fatalf("systemtest: unscripted call %s %s", c.Name, strings.Join(c.Args, " "))
	return &Call{}
}

func (r *Runner) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	call := r.answer(c)
	return call.output, call.err
}

func (r *Runner) Attach(_ context.Context, c local.Cmd, _ local.Stream) (local.Attached, error) {
	call := r.answer(c)
	last := ""
	for _, line := range strings.Split(call.output.Stderr, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			last = line
		}
	}
	return local.Attached{ExitCode: call.output.ExitCode, LastStderr: last}, call.err
}

func (r *Runner) Start(c local.Cmd, _ string) (int, error) {
	call := r.answer(c)
	if call.err != nil {
		return 0, call.err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pid := 1000 + len(r.Calls)
	r.live[pid] = true
	return pid, nil
}

func (r *Runner) LookPath(name string) (string, bool) {
	if search, ok := pathOf(r.Env); ok {
		return r.find(search, name)
	}
	path, ok := r.Paths[name]
	return path, ok
}

func (r *Runner) Within(env []string) local.Runner { return &within{r: r, env: env} }

func (r *Runner) find(search, name string) (string, bool) {
	for _, dir := range strings.Split(search, ":") {
		if candidate := dir + "/" + name; dir != "" && (slices.Contains(r.Programs, candidate) || r.Paths[name] == candidate) {
			return candidate, true
		}
	}
	return "", false
}

func pathOf(env []string) (string, bool) {
	search, set := "", false
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			search, set = v, true
		}
	}
	return search, set
}

// value: the Runner a child environment of its own gives: each call is
// recorded with that environment before its own Env, and a lookup walks
// that environment's PATH alone.
type within struct {
	r   *Runner
	env []string
}

func (w *within) at(c local.Cmd) local.Cmd {
	c.Env = append(append([]string{}, w.env...), c.Env...)
	return c
}

func (w *within) Output(ctx context.Context, c local.Cmd) (local.Output, error) {
	return w.r.Output(ctx, w.at(c))
}

func (w *within) Attach(ctx context.Context, c local.Cmd, out local.Stream) (local.Attached, error) {
	return w.r.Attach(ctx, w.at(c), out)
}

func (w *within) Start(c local.Cmd, logPath string) (int, error) { return w.r.Start(w.at(c), logPath) }

func (w *within) LookPath(name string) (string, bool) {
	search, _ := pathOf(w.env)
	return w.r.find(search, name)
}

func (w *within) Within(env []string) local.Runner { return &within{r: w.r, env: env} }
func (w *within) Alive(pid int) bool               { return w.r.Alive(pid) }
func (w *within) Signal(pid int, sig local.Sig) error {
	return w.r.Signal(pid, sig)
}
func (w *within) Reap(dir string) int { return w.r.Reap(dir) }

func (r *Runner) Alive(pid int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.live[pid]
}

func (r *Runner) Spawn(pid int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.live[pid] = true
}

func (r *Runner) Exit(pid int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.live, pid)
}

func (r *Runner) Signal(pid int, sig local.Sig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.live[pid] {
		return fmt.Errorf("no process %d", pid)
	}
	r.Signals = append(r.Signals, sig)
	delete(r.live, pid)
	return nil
}

func (r *Runner) Reap(string) int { return 0 }
