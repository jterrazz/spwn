package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageapprovals "github.com/jterrazz/studio-station/apps/engine/pages/approvals"
)

// value: the mini's approvals as its engine read them, with Spwn.app
// carrying the engine: nothing it can tell yet, Local Network refused
// with its fix, approved, and still checking.
func approvalsPages() []Fixture {
	of := func(read machines.LocalNetwork) page.Page {
		return pageapprovals.Page(approvals.ListingOf("mac-mini", read, true), thisMacContext())
	}
	return append(states("approvals",
		pending(of(machines.LocalNetworkApproved)),
		of(""),
		failure("approvals", "approvals", "the engine is not running", "station engine start"),
		of(machines.LocalNetworkNotApproved),
	), Fixture{"approvals.approved", of(machines.LocalNetworkApproved)}, Fixture{"approvals.checking", of(machines.LocalNetworkChecking)})
}
