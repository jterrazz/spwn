package calls

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/section"
)

type RemoveIn struct {
	Names   []string
	Confirm string
	// value: the owner asked this removal of these apps by name, and
	// confirmed it: only it takes a package station never recorded adding.
	// An apply's removal, decided by the ledger, leaves it false.
	Word bool
}

var Remove = call.Call[Deps, RemoveIn, []apps.Outcome]{
	Name: "apps.remove", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies, Confirm: confirm.Remove,
	Errors: []fail.Code{Unknown, Elsewhere, NotInstalled, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in RemoveIn, _ call.Progress) ([]apps.Outcome, error) {
		if err := d.Apps.CheckUninstall(in.Names); err != nil {
			return nil, coded(err)
		}
		if err := installed(ctx, d, in.Names); err != nil {
			return nil, err
		}
		if err := confirm.Require(confirm.For(confirm.Remove, d.Kind), strings.Join(in.Names, ", "), in.Confirm); err != nil {
			return nil, err
		}
		out, err := d.Apps.Uninstall(ctx, in.Names, in.Word)
		return interrupted(ctx, out, err)
	},
}
