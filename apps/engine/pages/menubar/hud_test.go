package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func saidWords(v page.Value) []string {
	out := make([]string, 0, len(v.Items))
	for _, w := range v.Items {
		out = append(out, w.Text)
	}
	return out
}

func hudWords(t *testing.T, p page.Page, id string) []string {
	t.Helper()
	a := actionOf(p, id)
	if a.Hud == nil {
		t.Fatalf("%s carries no HUD", id)
	}
	return saidWords(*a.Hud)
}

func TestTheHUDSaysWhatAGestureDidInTheWordsOfTheStories(t *testing.T) {
	workspace := macPageOf(t, "linux-3", nil)
	mac := macPageOf(t, miniMac, nil)
	cases := []struct {
		gesture string
		p       page.Page
		action  string
		want    []string
	}{
		{"Copy the footer's command", workspace, ActionCopySSH, []string{"Copied " + catalog.SSHCommand("", "linux-3")}},
		{"Open in Terminal, the footer's button", workspace, ActionOpenStation, []string{"linux-3 opened in Terminal"}},
		{"Attach from a Mac's row", mac, ActionAttach + "linux-3", []string{"linux-3 opened in Terminal"}},
	}
	for _, c := range cases {
		if got := hudWords(t, c.p, c.action); !slices.Equal(got, c.want) {
			t.Errorf("%s: the HUD says %q, want %q", c.gesture, got, c.want)
		}
	}
	if got := actionOf(workspace, ActionCopySSH).Run; got == nil || got.Copy != catalog.SSHCommand("", "linux-3") {
		t.Errorf("the footer's copy copies %+v, want its command", got)
	}
}

func TestATileSaysWhatItShowsAndItsShortcutInItsTooltip(t *testing.T) {
	w := newHeroWorld()
	w.r.Ports.Macs = []ports.MacPorts{{Name: heroPro, ThisMac: true, Shared: []ports.SharedPort{{VM: "linux-1", GuestPort: 5173}, {VM: "linux-1", GuestPort: 3000}}}}
	w.r.Awake = &awake.Report{Mac: heroPro, ThisMac: true, Engine: awake.EngineRunning, Zone: time.UTC, Session: &awake.Session{Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)}}
	p := homeOf(t, w)
	want := map[string]struct {
		words []string
		key   string
	}{
		"Ports":    {[]string{"2 shared"}, "cmd+p"},
		"Accounts": {[]string{"Accounts", "All machines"}, "cmd+u"},
		"Awake":    {[]string{"Awake until 18:00", "right-click for a duration"}, "cmd+e"},
	}
	for caption, w := range want {
		got := tile(t, p, caption)
		if !slices.Equal(saidWords(got.Tip), w.words) || got.Key != w.key {
			t.Errorf("%s: its tooltip says %q with %q, want %q with %q", caption, saidWords(got.Tip), got.Key, w.words, w.key)
		}
	}
	w.r.Awake.Session = nil
	if got := saidWords(tile(t, homeOf(t, w), "Awake").Tip); !slices.Equal(got, []string{"Awake", "right-click for a duration"}) {
		t.Errorf("off: its tooltip says %q", got)
	}
}
