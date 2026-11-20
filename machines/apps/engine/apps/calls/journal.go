package calls

import (
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
)

// invariant: an update keeps one line per app it moved or failed to, in
// the words of a machine's gesture ("node updated", "node not updated"),
// a failed one with the update that tries it again; an app it left as it
// was keeps none.
func keptUpdate(_ UpdateIn, out []apps.Outcome, _ error) []journal.Line {
	lines := make([]journal.Line, 0, len(out))
	for _, o := range out {
		l := journal.Line{Text: o.Name + " updated", Detail: o.Detail, Subject: &journal.Subject{Kind: journal.App, Name: o.Name}}
		switch o.Result {
		case catalog.StateBroken:
			l.Text, l.Outcome, l.Retry = o.Name+" not updated", journal.Failed, &finding.Fix{Call: "apps.update", Args: map[string]string{"Names": o.Name}}
		case catalog.StateOK:
			l.Outcome = journal.Done
		default:
			continue
		}
		lines = append(lines, l)
	}
	return lines
}
