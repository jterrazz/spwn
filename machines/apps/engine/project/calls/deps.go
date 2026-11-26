package calls

import (
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
)

type Deps struct {
	Projects contract.Projects
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Show, d), call.Bind(Create, d), call.Bind(Plan, d), call.Bind(Drift, d), call.Bind(Apply, d)}
}

const (
	NotFound       fail.Code = "project.not-found"
	Exists         fail.Code = "project.exists"
	BadName        fail.Code = "project.bad-name"
	UnknownHarness fail.Code = "project.unknown-harness"
	UnknownExample fail.Code = "project.unknown-example"
	// value: create --from with --harness: an example declares the harness
	// of each of its agents, so none is chosen for it.
	ExampleDeclaresHarness fail.Code = "project.example-declares-harness"
	// value: a plan that finds an error: the leaf refuses once the plan is
	// printed, so the owner reads every issue.
	Invalid fail.Code = "project.invalid"
	// value: a link where a call reaches: in the declaration, on a folder
	// of a run folder's path or on its record; refused before anything is
	// read or written through it, never followed.
	LinkInTheWay fail.Code = "project.link-in-the-way"
	// value: a name of the declaration that is neither a file nor a folder,
	// a FIFO say: refused before any of the declaration is read.
	NotAFile fail.Code = "project.not-a-file"
	// value: a change apply could not make, a folder in a rendered file's
	// way say: it stops there, recording what it wrote before.
	NotWritten fail.Code = "project.not-written"
)

func coded(err error, subject string) error {
	// invariant: a refusal of the Projects travels as its code, worded by
	// the binary that prints it.
	var at *project.Unwritten
	switch {
	case err == nil:
		return nil
	case errors.As(err, &at) && errors.Is(at.Err, project.ErrLink):
		return fail.New(fail.Refused, LinkInTheWay, at.Path, nil)
	case errors.As(err, &at) && errors.Is(at.Err, project.ErrNotAFile):
		return fail.New(fail.Refused, NotAFile, at.Path, nil)
	case errors.As(err, &at) && unrecorded(err):
		return fail.New(fail.Refused, NotWritten, at.Path, map[string]string{"record": "not-written"})
	case errors.As(err, &at):
		return fail.New(fail.Refused, NotWritten, at.Path, nil)
	case errors.Is(err, project.ErrNotFound):
		return fail.New(fail.NotFound, NotFound, subject, nil)
	case errors.Is(err, project.ErrExists):
		return fail.New(fail.Refused, Exists, subject, nil)
	}
	return err
}

// value: err holds the record's own write refused, alone or joined to the
// change refused before it.
func unrecorded(err error) bool {
	var at *project.Unwritten
	if errors.As(err, &at) && at.Path == project.StateFolder+"/"+project.RecordName {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return slices.ContainsFunc(joined.Unwrap(), unrecorded)
	}
	return false
}
