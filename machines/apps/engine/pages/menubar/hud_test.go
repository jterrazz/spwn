package menubar

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
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
	ssh, err := catalog.SSHCommand("", "linux-3")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		gesture string
		p       page.Page
		action  string
		want    []string
	}{
		{"Copy the footer's command", workspace, ActionCopySSH, []string{"Copied " + ssh}},
		{"Open in Terminal, the footer's button", workspace, ActionOpenStation, []string{"linux-3 opened in Terminal"}},
		{"Attach from a Mac's row", mac, ActionAttach + "linux-3", []string{"linux-3 opened in Terminal"}},
	}
	for _, c := range cases {
		if got := hudWords(t, c.p, c.action); !slices.Equal(got, c.want) {
			t.Errorf("%s: the HUD says %q, want %q", c.gesture, got, c.want)
		}
	}
	if got := actionOf(workspace, ActionCopySSH).Run; got == nil || got.Copy != ssh {
		t.Errorf("the footer's copy copies %+v, want its command", got)
	}
}
