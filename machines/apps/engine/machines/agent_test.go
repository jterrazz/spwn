package machines

import "testing"

func TestAnEngineThatServesNoAgentNeedsYouWithTheGestureThatFreesIt(t *testing.T) {
	for _, c := range []struct {
		story     string
		agent     Agent
		text, fix string
	}{
		{"another process holds the socket", Agent{State: AgentHeld, PID: 9006, Executable: "/private/tmp/old/Spwn.app/Contents/MacOS/Spwn Engine", Free: "kill 9006"},
			"mac-mini's ssh agent socket is held by Spwn Engine (pid 9006), not its engine", "kill 9006"},
		{"a process no job runs holds it", Agent{State: AgentHeld, PID: 4242, Free: "kill 4242"},
			"mac-mini's ssh agent socket is held by pid 4242, not its engine", "kill 4242"},
		{"the agent failed to open", Agent{State: AgentFailed, Detail: "listen unix: permission denied"},
			"mac-mini's engine serves no ssh agent: listen unix: permission denied", "station engine start"},
	} {
		mac := MacView{Name: "mac-mini", ThisMac: true, Reachable: true, Engine: EngineView{State: EngineRunning, Agent: c.agent}}
		found := Findings(AllMachines{Macs: []MacView{mac}})
		if len(found) != 1 || found[0].Text != c.text || found[0].Fix.Command != c.fix {
			t.Errorf("%s: %+v, want %q fixed by %q", c.story, found, c.text, c.fix)
		}
	}
	for _, a := range []Agent{{State: AgentServed}, {}} {
		mac := MacView{Name: "mac-mini", ThisMac: true, Reachable: true, Engine: EngineView{State: EngineRunning, Agent: a}}
		if found := Findings(AllMachines{Macs: []MacView{mac}}); len(found) != 0 {
			t.Errorf("an agent read %q needs you: %+v", a.State, found)
		}
	}
}
