package calls

import (
	"errors"

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
)

func coded(err error, subject string) error {
	// invariant: a refusal of the Projects travels as its code, worded by
	// the binary that prints it.
	switch {
	case err == nil:
		return nil
	case errors.Is(err, project.ErrNotFound):
		return fail.New(fail.NotFound, NotFound, subject, nil)
	case errors.Is(err, project.ErrExists):
		return fail.New(fail.Refused, Exists, subject, nil)
	}
	return err
}
