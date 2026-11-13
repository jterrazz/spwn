package accounts

import (
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func Findings(r Report, m machine.Ref) []finding.Finding {
	// invariant: an account of this machine that is broken (an expired
	// sign-in, a refused token) is a Needs-you line, fixed by the owner's own
	// sign-in command; a VM's accounts are its own Overview's.
	var out []finding.Finding
	for _, mm := range r.Machines {
		if !mm.ThisMac {
			continue
		}
		for _, a := range mm.Accounts {
			if a.State != catalog.StateBroken {
				continue
			}
			text := a.Provider + " sign-in fails"
			if a.Reason == ReasonRefused {
				text = a.Provider + " sign-in expired"
			}
			out = append(out, finding.Finding{Machine: m, Section: section.Accounts, Severity: finding.Broken, Subject: a.Provider,
				Text: text, Fix: finding.Fix{Command: a.Hint}})
		}
	}
	return out
}
