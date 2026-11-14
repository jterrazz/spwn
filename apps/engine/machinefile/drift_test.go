package machinefile_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func line(pkg string, kind catalog.Kind, c machinefile.Condition, a machinefile.Action) machinefile.Line {
	return machinefile.Line{Step: machinefile.Step{Package: pkg, Kind: kind}, Seen: machinefile.Seen{Condition: c}, Action: a}
}

func chapter(t *testing.T, d machinefile.Drift, c machinefile.Chapter) machinefile.ChapterDrift {
	t.Helper()
	for _, got := range d.Chapters {
		if got.Chapter == c {
			return got
		}
	}
	t.Fatalf("no chapter %s in %+v", c, d.Chapters)
	return machinefile.ChapterDrift{}
}

func TestDriftCountsEachChapterAndWordsItsVerdicts(t *testing.T) {
	// invariant: of 43 apps, 40 present, one absent, one present whose
	// block was edited by hand and one changed manager differ, with the
	// extra; the setting set by hand differs in settings.
	p := machinefile.Plan{Name: "linux-dev"}
	for i := range 40 {
		p.Lines = append(p.Lines, line("app"+string(rune('a'+i%26))+string(rune('a'+i/26)), catalog.KindApp, machinefile.Present, machinefile.Keep))
	}
	p.Lines = append(p.Lines,
		line("ripgrep", catalog.KindApp, machinefile.Absent, machinefile.Install),
		line("zoxide", catalog.KindApp, machinefile.Present, machinefile.Keep),
		line("node", catalog.KindManager, machinefile.Changed, machinefile.Leave),
		line("macos-dock", catalog.KindSetting, machinefile.Changed, machinefile.Leave),
		line("macos-keyboard", catalog.KindSetting, machinefile.Present, machinefile.Keep),
	)
	p.Blocks = []machinefile.Block{
		{Package: "zoxide", ID: "zoxide", Condition: machinefile.Changed, Action: machinefile.Leave},
		{Package: "apppa", ID: "apppa", Condition: machinefile.Present, Action: machinefile.Keep},
	}
	p.Extra = []machinefile.ExtraLine{{Line: machinefile.Line{Step: machinefile.Step{Package: "jq", Kind: catalog.KindApp}, Seen: machinefile.Seen{Condition: machinefile.Present}, Action: machinefile.Leave}}}
	d := machinefile.DriftOf(p, nil)
	apps := chapter(t, d, machinefile.ChapterApps)
	if apps.Listed != 43 || apps.Matching != 40 || len(apps.Differing) != 4 {
		t.Errorf("apps: %d of %d match, %d differ; want 40 of 43, 4", apps.Matching, apps.Listed, len(apps.Differing))
	}
	if apps.Verdict != (verdict.Verdict{Tone: verdict.Warn, Sentence: "4 differences"}) {
		t.Errorf("apps verdict %+v", apps.Verdict)
	}
	settings := chapter(t, d, machinefile.ChapterSettings)
	if settings.Listed != 2 || settings.Matching != 1 || settings.Verdict.Sentence != "1 difference" {
		t.Errorf("settings: %+v", settings)
	}
	if d.Differences != 5 || d.Verdict != (verdict.Verdict{Tone: verdict.Warn, Sentence: "5 differences from linux-dev."}) {
		t.Errorf("drift: %d, %+v", d.Differences, d.Verdict)
	}
	if extra := apps.Differing[len(apps.Differing)-1]; extra.Package != "jq" || extra.Condition != machinefile.Extra {
		t.Errorf("the extra reads %+v; want jq, extra", extra)
	}
}

func TestDriftOfAMatchingMachineIsCalm(t *testing.T) {
	p := machinefile.Plan{Name: "linux-dev", Lines: []machinefile.Line{line("jq", catalog.KindApp, machinefile.Present, machinefile.Keep)}}
	d := machinefile.DriftOf(p, nil)
	if d.Differences != 0 || d.Verdict != (verdict.Verdict{Tone: verdict.OK, Sentence: "Matches linux-dev."}) {
		t.Errorf("drift: %d, %+v", d.Differences, d.Verdict)
	}
	if apps := chapter(t, d, machinefile.ChapterApps); apps.Verdict != (verdict.Verdict{Tone: verdict.OK, Sentence: "Matches"}) || apps.Matching != 1 {
		t.Errorf("apps: %+v", apps)
	}
	if d.Apply.Sentence != "Nothing to apply." {
		t.Errorf("apply: %q", d.Apply.Sentence)
	}
}

func TestDriftSaysWhatApplyWouldDo(t *testing.T) {
	for _, c := range []struct {
		name  string
		plan  machinefile.Plan
		words string
	}{
		{"two installs and one left", machinefile.Plan{Lines: []machinefile.Line{
			line("ripgrep", catalog.KindApp, machinefile.Absent, machinefile.Install),
			line("macos-dock", catalog.KindSetting, machinefile.Absent, machinefile.Set),
			line("node", catalog.KindApp, machinefile.Changed, machinefile.Leave),
		}}, "Apply will install 2 and leave 1."},
		{"every verb", machinefile.Plan{Lines: []machinefile.Line{
			line("ripgrep", catalog.KindApp, machinefile.Absent, machinefile.Install),
			line("tree", catalog.KindApp, machinefile.NotManaged, machinefile.Adopt),
			line("node", catalog.KindApp, machinefile.Changed, machinefile.Leave),
		}, Extra: []machinefile.ExtraLine{{Installed: machinefile.Installed{Method: "pkg"}, Line: machinefile.Line{Step: machinefile.Step{Package: "jq"}, Seen: machinefile.Seen{Condition: machinefile.Present, Method: "pkg"}, Action: machinefile.Remove}}}},
			"Apply will install 1, adopt 1, remove 1 and leave 1."},
		{"only what is left", machinefile.Plan{Lines: []machinefile.Line{line("node", catalog.KindApp, machinefile.Changed, machinefile.Leave)}}, "Nothing to apply."},
		{"none", machinefile.Plan{}, "Nothing to apply."},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := machinefile.DriftOf(c.plan, nil).Apply.Sentence; got != c.words {
				t.Errorf("%q; want %q", got, c.words)
			}
		})
	}
}

func TestDriftReadsAVMsPortsAndAccountsAgainstItsFile(t *testing.T) {
	// invariant: a port shared public that the file opens to private
	// differs, as does an account the file shares that has no copy and a
	// copy the file does not share; what matches counts as matching.
	access := machinefile.Access{
		Ports: []machinefile.PortHeld{
			{Port: 3000, Seen: catalog.AudiencePublic, Asked: catalog.AudiencePrivate},
			{Port: 5173, Seen: catalog.AudiencePrivate, Asked: catalog.AudiencePrivate},
		},
		Accounts: []machinefile.AccountHeld{
			{Account: "forge", Seen: false, Asked: true},
			{Account: "mail", Seen: true, Asked: true},
			{Account: "notes", Seen: true, Asked: false},
		},
	}
	d := machinefile.DriftOf(machinefile.Plan{Name: "linux-dev"}, &access)

	ports := chapter(t, d, machinefile.ChapterPorts)
	if ports.Listed != 2 || ports.Matching != 1 || len(ports.Differing) != 1 {
		t.Fatalf("ports: %+v", ports)
	}
	if got := ports.Differing[0]; got.Port != 3000 || got.Condition != machinefile.Changed || got.SeenTo != catalog.AudiencePublic || got.AskedTo != catalog.AudiencePrivate {
		t.Errorf("port 3000 reads %+v; want changed, public against private", got)
	}
	accounts := chapter(t, d, machinefile.ChapterAccounts)
	if accounts.Listed != 2 || accounts.Matching != 1 || len(accounts.Differing) != 2 {
		t.Fatalf("accounts: %+v", accounts)
	}
	if got := accounts.Differing[0]; got.Account != "forge" || got.Condition != machinefile.Absent {
		t.Errorf("forge reads %+v; want absent, shared by the file", got)
	}
	if got := accounts.Differing[1]; got.Account != "notes" || got.Condition != machinefile.Extra {
		t.Errorf("notes reads %+v; want extra", got)
	}
	if d.Differences != 3 || d.Verdict.Sentence != "3 differences from linux-dev." {
		t.Errorf("drift: %d, %q", d.Differences, d.Verdict.Sentence)
	}
}

func TestADriftWithoutItsMacsReadSaysSo(t *testing.T) {
	d := machinefile.DriftOf(machinefile.Plan{Name: "linux-dev"}, nil).AccessUnread()
	if len(d.Chapters) != 2 {
		t.Errorf("chapters %+v; want apps and settings alone", d.Chapters)
	}
	if want := "Matches linux-dev. Its ports and accounts are read from its Mac."; d.Verdict.Sentence != want {
		t.Errorf("%q; want %q", d.Verdict.Sentence, want)
	}
}
