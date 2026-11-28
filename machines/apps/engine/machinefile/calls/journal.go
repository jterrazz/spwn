package calls

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// invariant: an apply keeps one line, the file it applied and what it
// installed and removed, counted, then skillLines.
func keptApply(in ApplyIn, out machinefile.Applied, _ error) []journal.Line {
	file := out.Plan.Name
	if file == "" {
		file = in.Machinefile
	}
	installed := 0
	for _, d := range out.Done {
		if d.Installed {
			installed++
		}
	}
	var said []string
	if installed > 0 {
		said = append(said, strconv.Itoa(installed)+" installed")
	}
	if n := len(out.Removed); n > 0 {
		said = append(said, strconv.Itoa(n)+" removed")
	}
	return append([]journal.Line{{Text: file + " applied", Detail: strings.Join(said, ", ")}}, skillLines(out.SkillFailed)...)
}

// invariant: a run that could not write the machine's `spwn` skill keeps
// one more line, failed, with why, so the journal of a run no one watched,
// the night's, says it; a run that wrote it keeps none.
func skillLines(failed string) []journal.Line {
	if failed == "" {
		return nil
	}
	return []journal.Line{{Text: "spwn skill", Detail: "not written: " + failed, Outcome: journal.Failed}}
}

// invariant: an upgrade keeps one line per app it moved, held or failed
// to move, a failed one with the words its update failed with and the
// update that tries it again; one that
// moved nothing keeps one line that says so; then skillLines.
func keptUpgrade(in UpgradeIn, out machinefile.Upgraded, err error) []journal.Line {
	n := machinefile.NightOf(out, err, in.Machinefile, time.Time{})
	app := func(name string) *journal.Subject { return &journal.Subject{Kind: journal.App, Name: name} }
	var lines []journal.Line
	for _, m := range n.Moved {
		lines = append(lines, journal.Line{Text: m.App, Subject: app(m.App), Outcome: journal.Done, Upgrade: &journal.Upgrade{From: m.Was, To: m.Now}})
	}
	for _, w := range n.Waiting {
		lines = append(lines, journal.Line{Text: w.App, Subject: app(w.App), Outcome: journal.Held, Detail: w.Words()})
	}
	why := map[string]string{}
	for _, d := range out.Done {
		why[d.Line.Step.Package] = d.Detail
	}
	for _, name := range n.Failed {
		lines = append(lines, journal.Line{Text: name, Detail: why[name], Subject: app(name), Outcome: journal.Failed, Retry: &finding.Fix{Call: "apps.update", Args: map[string]string{"Names": name}}})
	}
	if len(lines) == 0 {
		lines = append(lines, journal.Line{Text: n.Machinefile + " upgraded", Detail: "nothing moved"})
	}
	return append(lines, skillLines(out.SkillFailed)...)
}
