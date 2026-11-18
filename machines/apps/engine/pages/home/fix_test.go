package home_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/pages/home"
)

func TestAFixPrintsAsTheCommandThatRunsIt(t *testing.T) {
	const self = "macbook-pro"
	cases := map[string]struct {
		need overview.Need
		want string
	}{
		"a server's restart on another Mac": {overview.Need{Call: "machines.restart", Args: map[string]string{"Name": "infra"}, On: "mac-mini", Confirm: "infra"},
			"station machines restart infra --confirm=infra --on mac-mini"},
		"a workspace's delete on this Mac": {overview.Need{Call: "machines.delete", Args: map[string]string{"VM": "agent-7f"}, On: self, Confirm: "agent-7f"},
			"station machines delete agent-7f --confirm=agent-7f"},
		"a manager's updates in a VM": {overview.Need{Call: "apps.update", Args: map[string]string{"Managers": "tools"}, On: "linux-1"},
			"station apps update --manager tools --on linux-1"},
		"a task's removal here":          {overview.Need{Call: "background.remove", Args: map[string]string{"Label": "com.x.y"}, On: self}, "station background remove com.x.y"},
		"a misplaced folder's move":      {overview.Need{Call: "disk.move", Args: map[string]string{"Folder": "~/Old"}, On: self}, "station disk move ~/Old"},
		"a sign-in again on another Mac": {overview.Need{Call: "accounts.login", Args: map[string]string{"Program": "wrangler"}, On: "mac-mini"}, "station accounts login wrangler --on mac-mini"},
		"an app's update":                {overview.Need{Call: "apps.update", Args: map[string]string{"Names": "node"}, On: self}, "station apps update node"},
	}
	for name, c := range cases {
		s := overview.Summary{Head: overview.Head{Name: self, Kind: "computer", ThisMac: true}, Needs: []overview.Need{c.need}}
		hints := home.Hints(s, []string{self}, true)
		if len(hints) != 1 || hints[0].Command != c.want {
			t.Errorf("%s: %+v, want %q", name, hints, c.want)
		}
	}
	read := home.Hints(overview.Summary{Needs: []overview.Need{{Call: "disk.move", Args: map[string]string{"Folder": "~/Old"}, On: "linux-1"}}}, nil, true)
	if len(read) != 1 || read[0].Command != "station disk move ~/Old" {
		t.Errorf("a reader that does not know its own name adds no --on: %+v", read)
	}
}
