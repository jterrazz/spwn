package machines

import "testing"

// invariant: MB-03: the menu bar's red ! is a server down: in a loop even
// while it runs, powered off from inside though it is wanted stopped,
// stopped while wanted running; a server stopped on purpose, a healthy one
// and a workspace are not.
func TestAServerIsDownWhenTheMenuBarSaysSo(t *testing.T) {
	looping := Restarts{Looping: true}
	cases := []struct {
		name string
		vm   VM
		down bool
	}{
		{"in a loop while running", VM{Lifetime: AlwaysOn, State: Running, Want: "running", Restarts: looping}, true},
		{"powered off from inside, wanted stopped", VM{Lifetime: AlwaysOn, State: Stopped, Want: "stopped", Detail: PoweredOffInside}, true},
		{"in a loop while stopped, wanted stopped", VM{Lifetime: AlwaysOn, State: Stopped, Want: "stopped", Restarts: looping}, true},
		{"stopped while wanted running", VM{Lifetime: AlwaysOn, State: Stopped, Want: "running"}, true},
		{"broken", VM{Lifetime: AlwaysOn, State: Broken}, true},
		{"stopped on purpose", VM{Lifetime: AlwaysOn, State: Stopped, Want: "stopped"}, false},
		{"running", VM{Lifetime: AlwaysOn, State: Running, Want: "running"}, false},
		{"a workspace stopped, wanted running", VM{Lifetime: Persistent, State: Stopped, Want: "running", Restarts: looping}, false},
	}
	for _, c := range cases {
		if got := Down(c.vm); got != c.down {
			t.Errorf("%s: down %v, want %v", c.name, got, c.down)
		}
	}
}
