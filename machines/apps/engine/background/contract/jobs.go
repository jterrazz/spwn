package contract

import (
	"context"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/background"
)

type Jobs interface {
	Present() bool
	Names() Names
	// invariant: every job but the operating system's own, in every scope
	// readable without root, sorted by label.
	List(ctx context.Context) ([]background.Job, error)
	Read(ctx context.Context, r Ref) Status
	// value: each label the domain lists, enabled or not; a label it does not
	// list is absent.
	Overrides(ctx context.Context, d Domain) map[string]bool
	// invariant: the definition is written only when it changed; the job is
	// not loaded by it.
	Install(ctx context.Context, j JobSpec) (changed bool, err error)
	Load(ctx context.Context, r Ref) error
	Start(ctx context.Context, r Ref) error
	Restart(ctx context.Context, r Ref) error
	Stop(ctx context.Context, r Ref) error
	// invariant: the job is stopped whatever its state, then its definition
	// deleted; a definition already gone is no failure.
	Remove(ctx context.Context, r Ref) error
}

type Names struct{ Manager, Command string }

type Domain struct {
	UID    int
	System bool
}

type Ref struct {
	Domain
	Label string
	File  string
}

type Status struct {
	Loaded bool
	State  string
}

type JobSpec struct {
	Label     string
	Program   []string
	Env       map[string]string
	AtLogin   bool
	KeepAlive bool
	Log       string
	File      string
}

type Failed struct {
	// value: the command line the job manager was asked to run.
	Argv   []string
	Exit   int
	Stderr string
	Stdout string
	// value: the run itself failed: the command is missing or timed out.
	Err error
	// value: the command line the owner types to read the job.
	See string
}

func (f *Failed) Error() string {
	if f.Err != nil {
		return f.Err.Error()
	}
	message := f.Verb() + " exited " + strconv.Itoa(f.Exit)
	if f.Stderr != "" {
		message += ": " + f.Stderr
	}
	return message
}

func (f *Failed) Unwrap() error { return f.Err }

func (f *Failed) Verb() string {
	if len(f.Argv) < 2 {
		return strings.Join(f.Argv, " ")
	}
	return f.Argv[0] + " " + f.Argv[1]
}
