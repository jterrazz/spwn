package menubar

import (
	"strings"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func judgedFleet(edit func(*model.AllMachines)) *model.AllMachines {
	f := macFleet()
	if edit != nil {
		edit(f)
	}
	j := model.Judged(*f)
	return &j
}

// invariant: a machine page's subtitle begins with its machine's standing,
// the word the engine sent and the signal its tone draws, never a word the
// page worked out itself.
func TestAMachinePagesSubtitleBeginsWithItsStandingWordAndSignal(t *testing.T) {
	fleet := judgedFleet(func(f *model.AllMachines) {
		vms := f.Macs[1].VMs
		vms[0].State = model.Stopped
		vms[2].State = model.Broken
	})
	cases := []struct {
		name string
		word string
		sig  page.Signal
	}{
		{proMac, "online", page.SignalWell},
		{airMac, "asleep", page.SignalQuiet},
		{"linux-3", "stopped", page.SignalQuiet},
		{"review-bot", "running", page.SignalWell},
		{"infra", "down", page.SignalFailing},
	}
	for _, c := range cases {
		p := macPageOf(t, c.name, func(m *Machine) { m.All = fleet })
		summary := sectionOf(p, SectionSummary)
		if summary == nil || len(summary.Caption) == 0 {
			t.Fatalf("%s: no summary caption in %+v", c.name, p.Sections)
		}
		first := summary.Caption[0]
		if first.Text != c.word || first.Signal != c.sig {
			t.Errorf("%s: subtitle starts %q (%s), want %q (%s)", c.name, first.Text, first.Signal, c.word, c.sig)
		}
	}
}

// invariant: every subpage header says Back, whatever page it returns to, and
// carries the title its H1 draws; the page contract requires a title on its
// summary section, which says the same words, and the app draws the header's.
func TestEverySubpageHeaderSaysBackAndCarriesItsTitle(t *testing.T) {
	fleet := judgedFleet(nil)
	here := page.Context{Machine: proMac, Kind: "computer"}
	pages := map[string]page.Page{
		"This Mac":   macPageOf(t, proMac, func(m *Machine) { m.All = fleet }),
		"infra":      macPageOf(t, "infra", func(m *Machine) { m.All = fleet }),
		"linux-3":    macPageOf(t, "linux-3", func(m *Machine) { m.All = fleet }),
		"Accounts":   Accounts(signInsOfTwo(t), AccountsOf{Asks: yesNo}, here),
		"Keep awake": awakeOf(t, nil),
		"Ports":      fleetPorts(t, portFleet(), PortsOf{}),
		"Needs you":  NeedsPage(needsFleet(), "", here),
	}
	pages["Needs you, a Mac's"] = NeedsPage(needsFleet(), needsMini, here)
	for name, p := range pages {
		title := strings.TrimSuffix(name, ", a Mac's")
		if p.Header == nil || p.Header.Back != "Back" || p.Header.Title != title {
			t.Errorf("%s: header %+v, want Back %q and Title %q", name, p.Header, "Back", title)
			continue
		}
		if sum := sectionOf(p, SectionSummary); sum != nil && sum.Title != title {
			t.Errorf("%s: summary section titled %q, want the header's %q", name, sum.Title, title)
		}
	}
}

// invariant: a Mac's page that sleeps is a subpage too: its header says
// Back and carries the Mac's name.
func TestAnAsleepMacPageCarriesItsTitle(t *testing.T) {
	p := macPageOf(t, airMac, func(m *Machine) { m.All = judgedFleet(nil) })
	if p.Header == nil || p.Header.Back != "Back" || p.Header.Title != airMac {
		t.Fatalf("header %+v, want Back and %q", p.Header, airMac)
	}
}

// value: a verdict's tone draws as its signal; a tone the engine does not
// define draws quiet.
func TestAToneDrawsAsItsSignal(t *testing.T) {
	cases := map[string]page.Signal{
		"ok":      page.SignalWell,
		"new":     page.SignalActive,
		"warn":    page.SignalWaits,
		"bad":     page.SignalFailing,
		"plain":   page.SignalQuiet,
		"unheard": page.SignalQuiet,
		"":        page.SignalQuiet,
	}
	for tone, want := range cases {
		if got := toneSignal(tone); got != want {
			t.Errorf("toneSignal(%q) = %s, want %s", tone, got, want)
		}
	}
}
