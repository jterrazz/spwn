package home_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/settings"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/background"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/overview"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	"github.com/jterrazz/studio-station/src/ports"
)

var update = flag.Bool("update", false, "rewrite the golden from the presenter")

func fixed() pagehome.Overview {
	c, err := catalog.Builtin()
	if err != nil {
		panic(err)
	}
	node, _ := c.App("node")
	return pagehome.Overview{
		GeneratedAt: time.Date(2026, 10, 31, 14, 0, 0, 0, time.UTC),
		Station:     overview.Station{Version: "v3.0.0", Commit: "abc1234", Engine: overview.Check{Name: "daemon", State: catalog.StateOK, Value: "running v3.0.0"}},
		Machine: overview.Machine{
			Name: "macbook", Registered: true, Kind: catalog.Mac, Hostname: "macbook",
			OS: "Darwin 27.0.0", Arch: "arm64", User: "jterrazz", Shell: "zsh",
		},
		System: overview.System{
			Uptime: "3d 4h", Battery: "92% charging", MemoryTotal: 34359738368,
			Reading: activity.Reading{Top: activity.Busiest{
				CPU:    []activity.Process{{PID: 1, Name: "x", Value: 12.5}},
				Memory: []activity.Process{{PID: 1, Name: "x", Value: 1048576}},
			}},
		},
		Environment: overview.Environment{
			Network:     []overview.Check{{Name: "local ip", State: catalog.StateOK, Value: "192.168.1.2"}},
			Ports:       []overview.Port{{PID: 1, Name: "node", Port: "3000"}},
			Protections: []overview.Check{{Name: "filevault", State: catalog.StateOK, Value: "on"}},
			Updates:     []overview.Check{{Name: "softwareupdate", State: catalog.StateDegraded, Value: "3 outdated"}},
		},
		Workspace: pagehome.Workspace{
			Disk: []overview.Disk{{Name: "homebrew", Bytes: 123}},
		},
		Apps: []apps.Report{{App: node, Method: node.Installs[0].Install.Method(), State: catalog.StateOK, Version: "24.11.0", Managed: true}},
		Settings: []settings.Report{{
			Item:  catalog.Setting{Name: "hushlogin", Category: catalog.CategoryTerminal, Kinds: []catalog.MachineKind{catalog.Mac, catalog.Linux, catalog.MacOS}},
			State: catalog.StateOK,
		}},
		Setup: overview.Setup{State: catalog.StateOK, Steps: []overview.Step{
			{Name: "name", State: catalog.StateOK, Detail: "macbook"},
			{Name: "zshrc", State: catalog.StateOK},
			{Name: "layout", State: catalog.StateOK},
		}},
		Identity:   []overview.Check{{Name: "git-email", State: catalog.StateOK, Value: "admin@jterrazz.com"}},
		Background: []background.Task{{Label: "ai.jterrazz.x", Group: background.GroupUnknown, State: catalog.StateOK, Run: background.Running, PID: 12, Scope: "user", Plist: "~/Library/LaunchAgents/ai.jterrazz.x.plist"}},
		Gateways:   []overview.Check{{Name: catalog.ServeRow, State: catalog.StateOK, Value: "on"}},
		Tailnet: ports.Status{
			State: catalog.StateOK, Connected: true, BackendState: "Running",
			Name: "macbook", DNSName: "macbook.tail1234.ts.net", IP: "100.64.0.1",
			Variant: "app-store", CLI: catalog.TailnetCLI,
		},
	}
}

func TestTheStatusEncodingIsPinned(t *testing.T) {
	body, err := json.MarshalIndent(pagehome.SnapshotContract(fixed()), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got := string(body) + "\n"
	golden := filepath.Join("testdata", "status.v1.golden.json")
	if *update {
		if err = os.WriteFile(golden, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("station.status/v1 encoding changed; a field is only ADDED within v1 (-update rewrites it).\ngot:\n%s\nwant:\n%s", got, want)
	}
}
