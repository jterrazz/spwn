package launchd

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const command = "launchctl"

type Options struct {
	Home string
	// value: the system root the machine-wide folders are read under.
	Root string
	// value: the user this station runs as: its session is the one a hint names.
	UID int
	// value: how long one call of the job manager may take; zero is no limit.
	Budget time.Duration
}

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
	LookPath(name string) (string, bool)
}

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

type Jobs struct {
	run   Runner
	files Files
	o     Options
}

var _ contract.Jobs = (*Jobs)(nil)

func New(run Runner, files Files, o Options) *Jobs {
	return &Jobs{run: run, files: files, o: o}
}

func (j *Jobs) Present() bool {
	_, ok := j.run.LookPath(command)
	return ok
}

func (j *Jobs) Names() contract.Names {
	return contract.Names{Manager: "launchd", Command: command}
}

func domainOf(d contract.Domain) string {
	if d.System {
		return "system"
	}
	return "gui/" + strconv.Itoa(d.UID)
}

func target(r contract.Ref) string { return domainOf(r.Domain) + "/" + r.Label }

func (j *Jobs) launchctl(ctx context.Context, args ...string) (local.Output, error) {
	return j.run.Output(ctx, local.Cmd{Name: command, Args: args, Timeout: j.o.Budget})
}

func (j *Jobs) see(r contract.Ref) string {
	shown := target(r)
	if !r.System && r.UID == j.o.UID {
		shown = "gui/$(id -u)/" + r.Label
	}
	return command + " print " + shown
}

func firstLine(s string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return first
}

func (j *Jobs) must(ctx context.Context, r contract.Ref, args ...string) error {
	out, err := j.launchctl(ctx, args...)
	if err == nil && out.ExitCode == 0 {
		return nil
	}
	return &contract.Failed{
		Argv: append([]string{command}, args...), Exit: out.ExitCode,
		Stderr: firstLine(out.Stderr), Stdout: firstLine(out.Stdout), Err: err, See: j.see(r),
	}
}

func (j *Jobs) Read(ctx context.Context, r contract.Ref) contract.Status {
	out, err := j.launchctl(ctx, "print", target(r))
	if err != nil || out.ExitCode != 0 {
		return contract.Status{}
	}
	s := contract.Status{Loaded: true}
	for _, line := range strings.Split(out.Stdout, "\n") {
		if state, ok := strings.CutPrefix(strings.TrimSpace(line), "state = "); ok {
			s.State = state
			break
		}
	}
	return s
}

func (j *Jobs) Overrides(ctx context.Context, d contract.Domain) map[string]bool {
	// outside: `launchctl print-disabled <domain>` prints one `"<label>" => <value>`
	// per override: enabled or disabled, true for disabled on older macOS; false
	// is not read as either.
	out, err := j.launchctl(ctx, "print-disabled", domainOf(d))
	if err != nil || out.ExitCode != 0 {
		return nil
	}
	overrides := map[string]bool{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		quoted, value, ok := strings.Cut(strings.TrimSpace(line), " => ")
		label, unquoted := strings.CutPrefix(quoted, `"`)
		label, closed := strings.CutSuffix(label, `"`)
		if !ok || !unquoted || !closed {
			continue
		}
		switch value {
		case "enabled":
			overrides[label] = true
		case "disabled", "true":
			overrides[label] = false
		}
	}
	return overrides
}

func (j *Jobs) Install(_ context.Context, spec contract.JobSpec) (bool, error) {
	definition := plist(spec)
	current, _ := j.files.ReadFile(spec.File)
	changed := string(current) != definition
	for _, dir := range []string{path.Dir(spec.File), path.Dir(spec.Log)} {
		if err := j.files.MkdirAll(dir, 0o700); err != nil {
			return false, err
		}
	}
	if !changed {
		return false, nil
	}
	if err := j.files.WriteFile(spec.File, []byte(definition), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func (j *Jobs) Load(ctx context.Context, r contract.Ref) error {
	return j.must(ctx, r, "bootstrap", domainOf(r.Domain), r.File)
}

func (j *Jobs) Start(ctx context.Context, r contract.Ref) error {
	return j.must(ctx, r, "kickstart", target(r))
}

func (j *Jobs) Restart(ctx context.Context, r contract.Ref) error {
	return j.must(ctx, r, "kickstart", "-k", target(r))
}

func (j *Jobs) Stop(ctx context.Context, r contract.Ref) error {
	return j.must(ctx, r, "bootout", target(r))
}

func (j *Jobs) Remove(ctx context.Context, r contract.Ref) error {
	_ = j.Stop(ctx, r)
	if err := j.files.Remove(r.File); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
