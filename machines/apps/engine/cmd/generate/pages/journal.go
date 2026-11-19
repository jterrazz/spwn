package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	pagejournal "github.com/jterrazz/studio-station/apps/engine/pages/journal"
)

// value: what both Macs kept: last night's upgrade of node on each and a
// failed ripgrep on the mini, then a VM deleted this morning by a session.
func journalLines() []journal.Line {
	night := keepAwakeNow.Add(-8 * time.Hour)
	app := func(name string) *journal.Subject { return &journal.Subject{Kind: journal.App, Name: name} }
	return []journal.Line{
		{At: night, Machine: "macbook-pro", Call: "machinefile.upgrade", Text: "node", Subject: app("node"), By: "the night", Outcome: journal.Done, Upgrade: &journal.Upgrade{From: "22.11.0", To: "22.12.0"}, Night: night},
		{At: night, Machine: "mac-mini", Call: "machinefile.upgrade", Text: "node", Subject: app("node"), By: "the night", Outcome: journal.Done, Upgrade: &journal.Upgrade{From: "22.11.0", To: "22.12.0"}, Night: night},
		{At: night.Add(time.Minute), Machine: "mac-mini", Call: "machinefile.upgrade", Text: "ripgrep", Detail: "download timed out", Subject: app("ripgrep"), By: "the night", Outcome: journal.Failed, Night: night},
		{At: keepAwakeNow.Add(-90 * time.Minute), Machine: "gate-2", Call: "machines.delete", Text: "gate-2 deleted", Detail: "8 GB freed",
			Subject: &journal.Subject{Kind: journal.Machine, Name: "gate-2", Gone: true}, By: "review-bot", Outcome: journal.Done},
	}
}

func journalReading() journal.Reading {
	groups, next := journal.Read(journalLines(), keepAwakeNow, journal.Query{})
	return journal.Reading{Groups: groups, Next: next}
}

func journalPages() []Fixture {
	return states("journal",
		pending(pagejournal.Read(journalReading(), thisMacContext())),
		pagejournal.Read(journal.Reading{Empty: "Nothing was done in the last 7 days."}, thisMacContext()),
		failure("journal", "journal", "the journal on mac-mini cannot be read", "station journal --on mac-mini"),
		pagejournal.Read(journalReading(), thisMacContext()),
	)
}
