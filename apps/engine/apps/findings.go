package apps

import (
	"cmp"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func Findings(reports []Report, m machine.Ref) []finding.Finding {
	// invariant: Apps' Needs-you lines: each failed app broken; the updates
	// waiting need a look, one line per manager, in the order its first app
	// is listed, fixed by updating that manager's apps.
	var out []finding.Finding
	waiting := map[string]int{}
	var order []string
	for _, r := range reports {
		switch StateOf(r) {
		case Failed:
			out = append(out, finding.Finding{Machine: m, Section: section.Apps, Severity: finding.Broken, Subject: r.App.Name,
				Text: r.App.Name + " failed", Fix: finding.Fix{Call: "apps.update", Args: map[string]string{"name": r.App.Name}}})
		case UpdateAvailable:
			manager := cmp.Or(r.Manager, string(r.Method))
			if waiting[manager] == 0 {
				order = append(order, manager)
			}
			waiting[manager]++
		case NotInstalled, NotManaged, OK, NotKnown, Pending:
		}
	}
	for _, manager := range order {
		text := strconv.Itoa(waiting[manager]) + " " + manager + " updates waiting"
		if waiting[manager] == 1 {
			text = "1 " + manager + " update waiting"
		}
		args := map[string]string{"manager": manager}
		out = append(out, finding.Finding{Machine: m, Section: section.Apps, Severity: finding.NeedsALook, Subject: manager,
			Text: text, Fix: finding.Fix{Call: "apps.update", Args: args}})
	}
	return out
}
