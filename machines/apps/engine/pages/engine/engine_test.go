package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAStartThatLostTheMenuBarSaysSoWithItsFix(t *testing.T) {
	left := engine.MenuBarLeft{Why: "not shown again: SMAppServiceErrorDomain 1", Fix: "switch Show in menu bar on in Spwn's Settings"}
	for _, c := range []struct {
		name   string
		report engine.StartReport
		row    bool
	}{
		{"the menu bar restarted", engine.StartReport{Version: "v3.1.0", Installed: engine.Installed{Restarted: true, From: "v3.0.0"}}, false},
		{"the menu bar left off", engine.StartReport{Version: "v3.1.0", Installed: engine.Installed{Restarted: true, From: "v3.0.0", MenuBar: left}}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := Start(c.report, page.Context{})
			var found *page.Row
			for _, s := range p.Sections {
				for _, it := range s.Items {
					if it.Row != nil && it.Row.ID == MenuBarRow {
						found = it.Row
					}
				}
			}
			if (found != nil) != c.row {
				t.Fatalf("the menu bar's row is there: %v, want %v", found != nil, c.row)
			}
			if found == nil {
				return
			}
			raw, err := json.Marshal(found)
			if err != nil {
				t.Fatal(err)
			}
			if found.Mark != page.Attention || !strings.Contains(string(raw), left.Why) || !strings.Contains(string(raw), left.Fix) {
				t.Errorf("the menu bar's row = %s, want it marked for attention with %q and %q", raw, left.Why, left.Fix)
			}
		})
	}
}
