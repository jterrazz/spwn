package setup

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type Step struct {
	Name    string
	State   catalog.State
	Detail  string
	Log     string
	Changed bool
}

type Move struct {
	// value: one thing init takes out of the home root before its steps
	// run: From moves to To, one rename, or is deleted when To is empty, a
	// file its app writes again where its settings now point. Both as
	// shown (~/.bun/bin).
	From  string
	To    string
	Bytes int64
}

type Report struct {
	Steps []Step
	// value: what init moves, announced before anything moves; empty when
	// nothing is to move.
	Moves []Move
}
