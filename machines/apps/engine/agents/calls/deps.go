package calls

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
)

type Deps struct {
	Minds contract.Minds
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
	return []call.Handler{
		call.Bind(List, d), call.Bind(Show, d), call.Bind(Create, d), call.Bind(Edit, d), call.Bind(Delete, d),
		call.Bind(Fork, d), call.Bind(Export, d), call.Bind(Import, d), call.Bind(Dream, d), call.Bind(Sleep, d),
	}
}

const (
	NotFound       fail.Code = "agents.not-found"
	Exists         fail.Code = "agents.exists"
	BadName        fail.Code = "agents.bad-name"
	UnknownHarness fail.Code = "agents.unknown-harness"
	BadArchive     fail.Code = "agents.bad-archive"
	// value: an edit that asks no field.
	NothingAsked fail.Code = "agents.nothing-asked"
	// value: a project this machine holds that the target does not share.
	NotShared fail.Code = "agents.not-shared"
)

func coded(err error, name string) error {
	// invariant: a refusal of the Minds travels as its code, worded by the
	// binary that prints it.
	switch {
	case err == nil:
		return nil
	case errors.Is(err, agents.ErrNotFound):
		return fail.New(fail.NotFound, NotFound, name, nil)
	case errors.Is(err, agents.ErrExists):
		return fail.New(fail.Refused, Exists, name, nil)
	case errors.Is(err, agents.ErrBadName):
		return fail.New(fail.Refused, BadName, name, nil)
	case errors.Is(err, agents.ErrUnknownHarness):
		return fail.New(fail.Refused, UnknownHarness, name, nil)
	case errors.Is(err, agents.ErrBadArchive):
		return fail.New(fail.Refused, BadArchive, name, nil)
	}
	return projectcalls.Refusal(err)
}

func named(name string) error {
	if !agents.ValidName(name) {
		return fail.New(fail.Refused, BadName, name, nil)
	}
	return nil
}

func checked(e agents.Edit) error {
	if err := e.Check(); err != nil {
		return coded(err, *e.Harness)
	}
	return nil
}
