package finding_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func TestNeedsYouListsBrokenFirstThenBySectionThenMacBeforeItsVMs(t *testing.T) {
	this := machine.Ref{Name: "jterrazz-macbook-pro"}
	mini := machine.Ref{Name: "jterrazz-mac-mini"}
	infra := machine.Ref{Name: "infra", On: mini.Name}
	linux1 := machine.Ref{Name: "linux-1", On: this.Name}
	line := func(m machine.Ref, s section.ID, sev finding.Severity, subject string) finding.Finding {
		return finding.Finding{Machine: m, Section: s, Severity: sev, Subject: subject}
	}
	read := []finding.Finding{
		line(this, section.Settings, finding.NeedsALook, "starship"),
		line(mini, section.Ports, finding.Broken, "airplay-5000"),
		line(linux1, section.Apps, finding.NeedsALook, "jq"),
		line(this, section.Apps, finding.NeedsALook, "node"),
		line(infra, section.Overview, finding.Broken, "health"),
		line(this, section.Accounts, finding.NeedsALook, "github"),
		line(mini, section.Overview, finding.Broken, "engine"),
		line(this, section.Settings, finding.NeedsALook, "firewall"),
	}

	got := make([]string, 0, len(read))
	for _, f := range finding.Rank(read) {
		got = append(got, f.Machine.Name+" "+f.Subject)
	}

	want := []string{
		"jterrazz-mac-mini engine",
		"infra health",
		"jterrazz-mac-mini airplay-5000",
		"jterrazz-macbook-pro node",
		"linux-1 jq",
		"jterrazz-macbook-pro starship",
		"jterrazz-macbook-pro firewall",
		"jterrazz-macbook-pro github",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Needs you ranks\n%v\nwant\n%v", got, want)
	}
	if read[0].Subject != "starship" {
		t.Error("ranking reordered the read it was given")
	}
}

func TestNeedsYouListsEachMacFollowedByItsOwnVMsThenTheNextMac(t *testing.T) {
	this := machine.Ref{Name: "jterrazz-macbook-pro"}
	mini := machine.Ref{Name: "jterrazz-mac-mini"}
	infra := machine.Ref{Name: "infra", On: mini.Name}
	linux1 := machine.Ref{Name: "linux-1", On: this.Name}
	line := func(m machine.Ref, subject string) finding.Finding {
		return finding.Finding{Machine: m, Section: section.Apps, Severity: finding.NeedsALook, Subject: subject}
	}
	read := []finding.Finding{
		line(this, "node"),
		line(infra, "curl"),
		line(mini, "fd"),
		line(linux1, "jq"),
	}

	got := make([]string, 0, len(read))
	for _, f := range finding.Rank(read) {
		got = append(got, f.Machine.Name+" "+f.Subject)
	}

	want := []string{
		"jterrazz-macbook-pro node",
		"linux-1 jq",
		"jterrazz-mac-mini fd",
		"infra curl",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Needs you ranks\n%v\nwant\n%v", got, want)
	}
}

func TestNeedsYouRanksRedThenWhatBlocksYouThenWhatExpiresSoonestFirst(t *testing.T) {
	mini := machine.Ref{Name: "mac-mini"}
	this := machine.Ref{Name: "macbook-pro"}
	at := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		read []finding.Finding
		want []string
	}{
		{"a lease, a wait, a down server and an exposed port", []finding.Finding{
			{Machine: machine.Ref{Name: "linux-1", On: this.Name}, Section: section.Accounts, Urgency: finding.Expiring, Ends: at.Add(24 * time.Minute), Subject: "token"},
			{Machine: machine.Ref{Name: "review-bot", On: mini.Name}, Section: section.Overview, Urgency: finding.Blocking, Subject: "review-bot"},
			{Machine: this, Section: section.Ports, Severity: finding.Broken, Urgency: finding.Red, Subject: "5000"},
			{Machine: machine.Ref{Name: "infra", On: mini.Name}, Section: section.Overview, Severity: finding.Broken, Urgency: finding.Red, Subject: "infra"},
		}, []string{"infra", "5000", "review-bot", "token"}},
		{"two reds are never merged, each keeps its row", []finding.Finding{
			{Machine: machine.Ref{Name: "infra", On: mini.Name}, Section: section.Overview, Severity: finding.Broken, Urgency: finding.Red, Subject: "infra"},
			{Machine: machine.Ref{Name: "db", On: mini.Name}, Section: section.Overview, Severity: finding.Broken, Urgency: finding.Red, Subject: "db"},
		}, []string{"infra", "db"}},
		{"leases by their deadline, whatever the Mac", []finding.Finding{
			{Machine: machine.Ref{Name: "linux-1", On: this.Name}, Section: section.Accounts, Urgency: finding.Expiring, Ends: at.Add(24 * time.Minute), Subject: "s2"},
			{Machine: machine.Ref{Name: "linux-3", On: mini.Name}, Section: section.Accounts, Urgency: finding.Expiring, Ends: at.Add(9 * time.Minute), Subject: "s1"},
			{Machine: machine.Ref{Name: "agent-7f", On: this.Name}, Section: section.Accounts, Urgency: finding.Expiring, Ends: at.Add(28 * time.Minute), Subject: "s3"},
		}, []string{"s1", "s2", "s3"}},
		{"a need before any upkeep, even a broken one", []finding.Finding{
			{Machine: this, Section: section.Background, Severity: finding.Broken, Subject: "login-item"},
			{Machine: machine.Ref{Name: "linux-1", On: this.Name}, Section: section.Accounts, Urgency: finding.Expiring, Ends: at, Subject: "token"},
		}, []string{"token", "login-item"}},
	} {
		got := make([]string, 0, len(c.read))
		for _, f := range finding.Rank(c.read) {
			got = append(got, f.Subject)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: ranked %v, want %v", c.name, got, c.want)
		}
	}
}

func TestUpkeepListsASecuritySettingThenWhatFailsThenSignInsThenUpdates(t *testing.T) {
	this := machine.Ref{Name: "macbook-pro"}
	mini := machine.Ref{Name: "mac-mini"}
	read := []finding.Finding{
		{Machine: this, Section: section.Apps, Severity: finding.NeedsALook, Subject: "updates"},
		{Machine: mini, Section: section.Accounts, Severity: finding.Broken, Subject: "sign-in-b"},
		{Machine: this, Section: section.Accounts, Severity: finding.Broken, Subject: "sign-in-a"},
		{Machine: this, Section: section.Background, Severity: finding.Broken, Subject: "login-item"},
		{Machine: this, Section: section.Disk, Severity: finding.NeedsALook, Subject: "~/Downloads"},
		{Machine: this, Section: section.Settings, Severity: finding.NeedsALook, Subject: "firewall"},
	}

	got := make([]string, 0, len(read))
	for _, f := range finding.RankUpkeep(read, func(f finding.Finding) finding.Finding { return f }) {
		got = append(got, f.Subject)
	}

	want := []string{"firewall", "login-item", "sign-in-a", "sign-in-b", "updates", "~/Downloads"}
	if !slices.Equal(got, want) {
		t.Errorf("Upkeep ranks\n%v\nwant\n%v", got, want)
	}
}
