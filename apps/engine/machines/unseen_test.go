package machines

import (
	"testing"
	"time"
)

func TestAMacIsToldInOneSentence(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	for name, c := range map[string]struct {
		mac  MacView
		want string
	}{
		"engine and awake": {MacView{ThisMac: true, Engine: EngineView{State: EngineRunning}, Awake: "awake until 18:00"}, "Engine running. Awake until 18:00."},
		"engine alone":     {MacView{Reachable: true, Engine: EngineView{State: EngineRunning}}, "Engine running."},
		"engine stopped":   {MacView{ThisMac: true, Engine: EngineView{State: EngineStopped}}, "Engine stopped."},
		"engine unknown":   {MacView{ThisMac: true}, ""},
		"not seen":         {MacView{Answered: now.Add(-12 * time.Minute)}, "Not seen for 12 min. Asleep or off the network."},
		"asleep":           {MacView{Answered: now.Add(-12 * time.Minute), Asleep: true}, "Asleep, not seen for 12 min."},
		"never seen":       {MacView{}, "Never seen. Asleep or off the network."},
	} {
		if got := SentenceOf(c.mac, now); got != c.want {
			t.Errorf("%s: %q; want %q", name, got, c.want)
		}
	}
}
