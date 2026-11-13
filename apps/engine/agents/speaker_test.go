package agents_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

func TestAVMIsSaidByTheAgentWaitingOnYouThenAtWorkThenTheLastToFinish(t *testing.T) {
	at := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	done := func(s string, ended time.Time) agents.Agent {
		return agents.Agent{Session: s, State: agents.Finished, Ended: ended}
	}
	for _, c := range []struct {
		name string
		kept []agents.Agent
		want string
	}{
		{"one waits among others", []agents.Agent{done("a", at), {Session: "b", State: agents.Working}, {Session: "c", State: agents.Waiting}}, "c"},
		{"one at work beside finished ones", []agents.Agent{done("a", at), {Session: "b", State: agents.Working}}, "b"},
		{"all finished: the last to end", []agents.Agent{done("a", at), done("b", at.Add(time.Minute)), done("c", at.Add(-time.Minute))}, "b"},
	} {
		got, ok := agents.Speaker(c.kept)
		if !ok || got.Session != c.want {
			t.Errorf("%s: said by %q, want %q", c.name, got.Session, c.want)
		}
	}
	if _, ok := agents.Speaker(nil); ok {
		t.Error("a VM with no agent is said by one")
	}
}
