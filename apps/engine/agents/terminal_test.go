package agents_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

func TestAHookedAgentIsSaidByItsHookAndAnUnhookedOneByTheHost(t *testing.T) {
	hooked := agents.Agent{Session: "s-1", Terminal: "t1", App: "claudecode", State: agents.Working}
	live := []agents.Terminal{
		{Address: "t1", App: "claudecode", Seen: agents.SeenBlocked, Message: "Allow Bash?"},
		{Address: "t2", App: "reviewer", Folder: "/home/admin/api", Seen: agents.SeenBlocked, Message: "Claude needs your permission to use Bash\n```rm -rf /```"},
		{Address: "t3", App: "claudecode", Seen: agents.SeenWorking},
		{Address: "t4", App: "claudecode"},
		{Address: "mini/t1", App: "claudecode", Seen: agents.SeenWorking},
	}
	got := agents.Detected([]agents.Agent{hooked}, live)
	if len(got) != 4 {
		t.Fatalf("agents %+v, want the hooked one and three the host sees", got)
	}
	if got[0].State != agents.Working || got[0].Ask != nil {
		t.Errorf("hooked %+v, want it as its hook said", got[0])
	}
	if a := got[1]; a.Session != "t2" || a.Terminal != "t2" || a.State != agents.Waiting || a.Ask == nil || a.Ask.Title != "Claude needs your permission to use Bash" || strings.Contains(a.Ask.Detail, "rm") || a.Folder != "/home/admin/api" {
		t.Errorf("blocked %+v (%+v), want it waiting on the program's first line, no code kept", a, a.Ask)
	}
	if got[2].State != agents.Working || got[3].State != agents.Running {
		t.Errorf("states %s, %s; want working, then running for one that says nothing", got[2].State, got[3].State)
	}
}
