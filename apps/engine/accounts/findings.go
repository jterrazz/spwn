package accounts

import (
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func Findings(r ProgramsRead, m machine.Ref) []finding.Finding {
	// invariant: a sign-in of this machine that no longer works is a
	// Needs-you line, worded by its row's own line, fixed by signing in again.
	var out []finding.Finding
	for _, p := range r.Programs {
		if p.State != ProgramBroken {
			continue
		}
		out = append(out, finding.Finding{Machine: m, Section: section.Accounts, Severity: finding.Broken, Subject: string(p.Program),
			Text: string(p.Program) + " sign-in fails: " + p.Line,
			Fix:  finding.Fix{Call: "accounts.login", Args: map[string]string{"Program": string(p.Program)}}})
	}
	return out
}
