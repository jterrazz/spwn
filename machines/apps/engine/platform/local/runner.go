package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type runner struct {
	env     []string
	dirs    []string
	system  []string
	landing []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (r *runner) LookPath(name string) (string, bool) {
	if strings.Contains(name, "/") {
		return name, executable(name)
	}
	for _, dir := range r.dirs {
		candidate := filepath.Join(dir, name)
		if executable(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func (r *runner) Within(env []string) Runner {
	path := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	within := *r
	within.env, within.dirs = env, searched(path, r.system, r.landing)
	return &within
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func (r *runner) command(ctx context.Context, c Cmd) (context.Context, *exec.Cmd, context.CancelFunc, error) {
	path, ok := r.LookPath(c.Name)
	if !ok {
		return ctx, nil, nil, fmt.Errorf("%s: %w", c.Name, ErrNotFound)
	}
	cancel := context.CancelFunc(func() {})
	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
	}
	cmd := exec.CommandContext(ctx, path, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(append([]string{}, r.env...), c.Env...)
	if !c.Sealed {
		cmd.Env = r.withLanding(cmd.Env)
	}
	if c.Track != "" {
		// invariant: a tracked child leads its own process group, so the end of
		// its context kills whatever it spawned too, never only the direct child.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
		cmd.WaitDelay = time.Second
	}
	return ctx, cmd, cancel, nil
}

// invariant: a child finds by name what an earlier step of the run installed: the
// landing folders that exist now are appended to the PATH it is given, the last
// PATH of env, so a PATH set for one command keeps its order and gains them last.
func (r *runner) withLanding(env []string) []string {
	path := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	have := map[string]bool{}
	for _, d := range strings.Split(path, ":") {
		have[d] = true
	}
	for _, d := range r.landing {
		if info, err := os.Stat(d); err == nil && info.IsDir() && !have[d] {
			if path != "" {
				path += ":"
			}
			path += d
			have[d] = true
		}
	}
	if path != "" {
		env = append(env, "PATH="+path)
	}
	return env
}

func run(cmd *exec.Cmd, track string) error {
	if track == "" {
		return cmd.Run()
	}
	if err := os.MkdirAll(track, 0o700); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// invariant: a tracked child's file lives while it runs, so a parent killed
	// with -9 leaves the record its next start reaps.
	file := filepath.Join(track, strconv.Itoa(cmd.Process.Pid))
	if start, err := StartOf(cmd.Process.Pid); err == nil {
		_ = os.WriteFile(file, []byte(start+"\n"), 0o600)
	}
	err := cmd.Wait()
	_ = os.Remove(file)
	return err
}

func (r *runner) Reap(dir string) int {
	// invariant: a recorded pid is killed only while it still names the
	// process that was recorded: its start, read from the kernel as
	// Processes reads it, must be the one written, so a recycled pid
	// survives.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	killed := 0
	for _, e := range entries {
		file := filepath.Join(dir, e.Name())
		data, _ := os.ReadFile(file)
		_ = os.Remove(file)
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 1 {
			continue
		}
		if !(Processes{}).Runs(pid, strings.TrimSpace(string(data))) {
			continue
		}
		if syscall.Kill(-pid, syscall.SIGKILL) == nil {
			killed++
		}
	}
	return killed
}

func (r *runner) Output(ctx context.Context, c Cmd) (Output, error) {
	ctx, cmd, cancel, err := r.command(ctx, c)
	if err != nil {
		return Output{}, err
	}
	defer cancel()
	if c.Timeout > 0 && c.Track == "" {
		bounded(cmd)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	code, err := exitOf(ctx, run(cmd, c.Track))
	return Output{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}, err
}

// invariant: a command read within a bound leads its own process group:
// past the bound the whole group is told to stop (SIGTERM), killed a second
// later (SIGKILL), and its pipes closed then, so nothing it started holds
// the read past its bound.
func bounded(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error {
		group := -cmd.Process.Pid
		time.AfterFunc(time.Second, func() { _ = syscall.Kill(group, syscall.SIGKILL) })
		return syscall.Kill(group, syscall.SIGTERM)
	}
}

func (r *runner) Attach(ctx context.Context, c Cmd, out Stream) (Attached, error) {
	ctx, cmd, cancel, err := r.command(ctx, c)
	if err != nil {
		return Attached{}, err
	}
	defer cancel()
	logs, err := appendTo(c.Logs)
	if err != nil {
		return Attached{}, err
	}
	defer logs.close()
	stderr := &lastLine{w: r.stderr}
	cmd.Stdout, cmd.Stderr = stderr, stderr
	switch {
	case len(logs.files) > 0:
		stderr.w = logs.writer()
	case out == ToStdout:
		cmd.Stdout = r.stdout
	}
	cmd.Stdin = r.stdin
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	var silent *answer
	if c.Answer > 0 {
		silent = awaitAnswer(cmd, c.Answer)
		defer silent.stop()
	}
	code, err := exitOf(ctx, run(cmd, c.Track))
	if silent.passed() {
		return Attached{ExitCode: -1, LastStderr: stderr.last()}, ErrSilent
	}
	return Attached{ExitCode: code, LastStderr: stderr.last()}, err
}

// invariant: a command that owes a first word within its bound leads its own
// process group, and the whole group is told to stop (SIGTERM, which sudo
// relays to what it runs as root) when the bound passes in silence; a
// grandchild that keeps the output open cannot hold the run past WaitDelay.
// Whatever the command says after its first word is free.
type answer struct {
	timer  *time.Timer
	heard  atomic.Bool
	silent atomic.Bool
	ended  atomic.Bool
}

func awaitAnswer(cmd *exec.Cmd, bound time.Duration) *answer {
	a := &answer{}
	cmd.Stdout = &heard{w: cmd.Stdout, a: a}
	cmd.Stderr = &heard{w: cmd.Stderr, a: a}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.WaitDelay = 5 * time.Second
	a.timer = time.AfterFunc(bound, func() {
		if a.heard.Load() || cmd.Process == nil {
			return
		}
		a.silent.Store(true)
		group := -cmd.Process.Pid
		_ = syscall.Kill(group, syscall.SIGTERM)
		time.AfterFunc(5*time.Second, func() {
			if !a.ended.Load() {
				_ = syscall.Kill(group, syscall.SIGKILL)
			}
		})
	})
	return a
}

func (a *answer) stop() {
	a.ended.Store(true)
	a.timer.Stop()
}

func (a *answer) passed() bool { return a != nil && a.silent.Load() }

type heard struct {
	w io.Writer
	a *answer
}

func (h *heard) Write(p []byte) (int, error) {
	if len(p) > 0 {
		h.a.heard.Store(true)
	}
	if h.w == nil {
		return len(p), nil
	}
	return h.w.Write(p)
}

type logFiles struct{ files []*os.File }

// invariant: a log is appended to, never truncated, and created owner-only
// with its folder, so a second run keeps what the first one said.
func appendTo(paths []string) (logFiles, error) {
	var l logFiles
	for _, p := range paths {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			l.close()
			return logFiles{}, err
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			l.close()
			return logFiles{}, err
		}
		l.files = append(l.files, f)
	}
	return l, nil
}

func (l logFiles) writer() io.Writer {
	ws := make([]io.Writer, 0, len(l.files))
	for _, f := range l.files {
		ws = append(ws, f)
	}
	return io.MultiWriter(ws...)
}

func (l logFiles) close() {
	for _, f := range l.files {
		_ = f.Close()
	}
}

type lastLine struct {
	w        io.Writer
	line     []byte
	nonEmpty string
}

func (l *lastLine) Write(p []byte) (int, error) {
	for _, b := range p {
		if b != '\n' {
			l.line = append(l.line, b)
			continue
		}
		l.keep()
	}
	return l.w.Write(p)
}

func (l *lastLine) keep() {
	if line := strings.TrimSpace(string(l.line)); line != "" {
		l.nonEmpty = line
	}
	l.line = l.line[:0]
}

func (l *lastLine) last() string {
	l.keep()
	return l.nonEmpty
}

func exitOf(ctx context.Context, err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return -1, ErrTimeout
		}
		return -1, ctxErr
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if interrupted(exit) {
			return -1, context.Canceled
		}
		return exit.ExitCode(), nil
	}
	return -1, err
}

// invariant: a child that Ctrl-C ended is an interruption, whether or not
// station's own handler has cancelled the context yet: the terminal tells the
// whole foreground group at once, so the child's exit can be reaped before the
// handler's goroutine runs, and that exit is never a failure of the command.
// The child either died of SIGINT or, as a shell does, said so with 128+SIGINT.
func interrupted(exit *exec.ExitError) bool {
	status, ok := exit.Sys().(syscall.WaitStatus)
	if !ok {
		return false
	}
	return status.Signaled() && status.Signal() == syscall.SIGINT || status.Exited() && status.ExitStatus() == 128+int(syscall.SIGINT)
}

func (r *runner) Start(c Cmd, logPath string) (int, error) {
	path, ok := r.LookPath(c.Name)
	if !ok {
		return 0, fmt.Errorf("%s: %w", c.Name, ErrNotFound)
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, err
	}
	defer func() { _ = log.Close() }()
	cmd := exec.Command(path, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(append([]string{}, r.env...), c.Env...)
	cmd.Stdout, cmd.Stderr = log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	// invariant: the child is reaped here when it ends, so a long-lived parent (the
	// daemon) never keeps a zombie that Alive would read as running.
	go func() { _ = cmd.Wait() }()
	return cmd.Process.Pid, nil
}

func (r *runner) Alive(pid int) bool {
	return pid > 0 && syscall.Kill(pid, 0) == nil
}

func (r *runner) Signal(pid int, sig Sig) error {
	s := syscall.SIGTERM
	if sig == SigKill {
		s = syscall.SIGKILL
	}
	return syscall.Kill(pid, s)
}
