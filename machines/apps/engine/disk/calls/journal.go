package calls

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

// invariant: a free keeps one line per row it reached, its own words on
// what was freed, a row that broke kept failed with its reason; a row it
// left alone keeps none.
func keptFreed(outcomes []disk.Outcome, subject journal.Kind) []journal.Line {
	lines := make([]journal.Line, 0, len(outcomes))
	for _, o := range outcomes {
		l := journal.Line{Text: o.Name + " freed", Detail: o.Detail}
		if subject != "" {
			l.Subject = &journal.Subject{Kind: subject, Name: o.Name}
		}
		switch {
		case o.Result == catalog.StateBroken:
			l.Text, l.Outcome, l.Detail = o.Name+" not freed", journal.Failed, o.Reason
		case o.Freed == 0:
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

func keptFree(_ FreeIn, out []disk.Outcome, _ error) []journal.Line {
	return keptFreed(out, journal.App)
}

func keptFreeUnused(_ FreeUnusedIn, out []disk.Outcome, _ error) []journal.Line {
	return keptFreed(out, "")
}
