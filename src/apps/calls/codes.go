package calls

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
)

const (
	Unknown      fail.Code = "apps.unknown"
	Elsewhere    fail.Code = "apps.lives-elsewhere"
	NotInstalled fail.Code = "apps.not-installed"
)

func coded(err error) error {
	// invariant: the refusals of the Apps calls travel as their codes, worded
	// by the binary that prints them.
	var unknown *apps.UnknownError
	var elsewhere *apps.ElsewhereError
	var absent *apps.NotInstalledError
	switch {
	case errors.As(err, &unknown):
		return fail.New(fail.NotFound, Unknown, unknown.Name, nil)
	case errors.As(err, &elsewhere):
		return fail.New(fail.Refused, Elsewhere, elsewhere.App, map[string]string{"lives": apps.LivesOn(elsewhere.Kinds), "vm": elsewhere.VM})
	case errors.As(err, &absent):
		return fail.New(fail.Refused, NotInstalled, absent.App, nil)
	}
	return err
}

func step(p call.Progress, task string) {
	if p != nil {
		p.Step(task, "")
	}
}

func updating(in UpdateIn) string {
	switch {
	case len(in.Names) > 0:
		return strings.Join(in.Names, ", ")
	case len(in.Method) > 0:
		methods := make([]string, 0, len(in.Method))
		for _, m := range in.Method {
			methods = append(methods, string(m))
		}
		return "the " + strings.Join(methods, ", ") + " apps"
	}
	return "the apps with an update"
}

func interrupted(ctx context.Context, out []apps.Outcome, err error) ([]apps.Outcome, error) {
	// invariant: an interrupted run is the interrupt, never a list of
	// failures: what finished stays done, and nothing is printed after it.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, coded(err)
}
