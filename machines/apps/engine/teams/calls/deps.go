package calls

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/teams"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
)

type Deps struct {
	Teams contract.Teams
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
	return []call.Handler{call.Bind(List, d), call.Bind(Show, d)}
}

const NotFound fail.Code = "teams.not-found"

func coded(err error, slug string) error {
	// invariant: a refusal of the Teams travels as its code, worded by the
	// binary that prints it.
	if errors.Is(err, teams.ErrNotFound) {
		return fail.New(fail.NotFound, NotFound, slug, nil)
	}
	return err
}
