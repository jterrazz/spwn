package calls

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

type Deps struct {
	Automations contract.Automations
	// value: the home of the user the target runs as, which a project
	// root of `~/` is read from.
	Home string
}

func (d Deps) root(project string) string {
	if rest, ok := strings.CutPrefix(project, "~/"); ok {
		return filepath.Join(d.Home, rest)
	}
	return project
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Show, d), call.Bind(Log, d)}
}

const (
	NotFound       fail.Code = "automations.not-found"
	NotUnique      fail.Code = "automations.not-unique"
	InvalidProject fail.Code = "automations.invalid-project"
)

func coded(err error, project, name, verb string) error {
	// invariant: a refusal of the Automations travels as its code, worded by
	// the binary that prints it; verb is the command the hint is typed with,
	// and a manifest that does not read is named by its project's own path.
	var many *automations.NotUniqueError
	var bad *automations.InvalidProjectError
	switch {
	case errors.Is(err, automations.ErrNotFound):
		return fail.New(fail.NotFound, NotFound, name, nil)
	case errors.As(err, &many):
		return fail.New(fail.Refused, NotUnique, name, map[string]string{"groups": strings.Join(many.Groups, ", "), "first": many.Groups[0], "verb": verb})
	case errors.As(err, &bad):
		return fail.New(fail.Refused, InvalidProject, project+"/spwn.yaml", map[string]string{"error": bad.Reason})
	}
	return err
}
