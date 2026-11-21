package readers

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// value: the Go value each family's reader puts, which a copy of another
// Mac's slice is decoded into.
type (
	macView        = machines.MacView
	macPorts       = ports.MacPorts
	overviewKept   = overview.Kept
	awakeReport    = awake.Report
	activityReport = activity.Report
	signinsRead    = accounts.ProgramsRead
)
