package calls

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
)

type Deps struct {
	Files contract.Files
	Kind  machine.Kind
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
	return []call.Handler{call.Bind(List, d), call.Bind(Show, d), call.Bind(Create, d), call.Bind(Delete, d)}
}

const (
	NotFound fail.Code = "skills.not-found"
	Exists   fail.Code = "skills.exists"
	BadName  fail.Code = "skills.bad-name"
)

func coded(err error, name string) error {
	// invariant: a refusal of the Files travels as its code, worded by the
	// binary that prints it.
	switch {
	case err == nil:
		return nil
	case errors.Is(err, skills.ErrNotFound):
		return fail.New(fail.NotFound, NotFound, name, nil)
	case errors.Is(err, skills.ErrExists):
		return fail.New(fail.Refused, Exists, name, nil)
	case errors.Is(err, skills.ErrBadName):
		return fail.New(fail.Refused, BadName, name, nil)
	}
	return err
}
