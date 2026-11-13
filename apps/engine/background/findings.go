package background

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func Findings(tasks []Task, m machine.Ref) []finding.Finding {
	// invariant: a failing task is broken, an orphan needs a look; each is
	// fixed by its removal, which station runs in the user scope and the
	// owner runs by hand in the root one (never sudo).
	out := make([]finding.Finding, 0, len(tasks))
	for _, t := range tasks {
		f := finding.Finding{Machine: m, Section: section.Background, Subject: t.Label, Fix: FixOf(t)}
		switch {
		case t.Run == Failing:
			f.Severity, f.Text = finding.Broken, t.Label+" is failing, last exit "+strconv.Itoa(t.Exit)
		case t.Group == GroupOrphan:
			f.Severity, f.Text = finding.NeedsALook, t.Label+" is left over"
		default:
			continue
		}
		out = append(out, f)
	}
	return out
}

func FixOf(t Task) finding.Fix {
	switch {
	case t.Group == GroupStation:
		return finding.Fix{}
	case t.Scope == ScopeRoot:
		return finding.Fix{Command: t.ByHand}
	}
	return finding.Fix{Call: "background.remove", Args: map[string]string{"name": t.Label}}
}
