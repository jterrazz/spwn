package architecture_test

import (
	"bytes"
	"encoding/json"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"os"
	"path/filepath"
	"testing"

	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/src/background/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"
)

// value: every call of the binary, bound with zero Deps: its Meta only.
func allCalls(t *testing.T) *call.Registry {
	t.Helper()
	registry, err := call.NewRegistry(
		accountscalls.All(accountscalls.Deps{}), appscalls.All(appscalls.Deps{}), awakecalls.All(awakecalls.Deps{}), backgroundcalls.All(backgroundcalls.Deps{}),
		diskcalls.All(diskcalls.Deps{}), imagescalls.All(imagescalls.Deps{}), machinescalls.All(machinescalls.Deps{}),
		portscalls.All(portscalls.Deps{}), settingscalls.All(settingscalls.Deps{}), setupcalls.All(setupcalls.Deps{}),
		activitycalls.All(activitycalls.Deps{}), enginecalls.All(enginecalls.Deps{}), overviewcalls.All(overviewcalls.Sections{}), agentscalls.All(agentscalls.Deps{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

// invariant: rule calls-json-fresh (13.3 rule 14, AR-08): calls.json is the
// registry's Meta as make generate writes it, so a call added, renamed or
// reshaped without make generate fails here.
func TestCallsJSONFresh(t *testing.T) {
	want, err := allCalls(t).Catalogue()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repoRoot(t), "src", "remote", "api", "calls.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("src/remote/api/calls.json is not the registry's catalogue: run make generate and commit the result")
	}
}

// invariant: rule palette-covers-calls (13.3 rule 14, MB-08): every action a
// page fixture carries (the menu bar's and the TUI's, the stories' pages,
// the palette's entries once it lands) names a call of calls.json, or opens
// a terminal, or copies or opens what its page gave it; nothing else runs.
func TestPaletteCoversCalls(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "src", "remote", "api", "calls.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c call.Catalogue
	if err = json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, e := range c.Calls {
		known[e.Name] = true
	}
	var files []string
	for _, dir := range []string{filepath.Join(root, "src", "ui", "testdata", "pages"), filepath.Join(root, "apps", "menubar", "stories", "pages")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	called := 0
	for _, file := range files {
		name := filepath.Base(file)
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var p struct {
			Actions []struct {
				ID  string `json:"id"`
				Run *struct {
					Call     string `json:"call"`
					Terminal bool   `json:"terminal"`
					Copy     string `json:"copy"`
					Open     string `json:"open"`
				} `json:"run"`
			} `json:"actions"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, a := range p.Actions {
			switch {
			case a.Run == nil, a.Run.Call == "" && (a.Run.Terminal || a.Run.Copy != "" || a.Run.Open != ""):
			case !known[a.Run.Call]:
				t.Errorf("%s: the action %s names %q, no call of calls.json", name, a.ID, a.Run.Call)
			default:
				called++
			}
		}
	}
	if called == 0 {
		t.Error("no page fixture carries an action that names a call: the rule reads nothing")
	}
}
