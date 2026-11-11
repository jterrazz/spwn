package ports

import "testing"

func TestEachPortIsAnnouncedOnceIPv4First(t *testing.T) {
	got := Announce([]Listener{
		{Port: 8081, Bind: "::", Process: "node", PID: 412}, {Port: 8081, Bind: "0.0.0.0", Process: "node", PID: 412},
		{Port: 8081, Bind: "::1", Process: "node", PID: 412}, {Port: 5432, Bind: "127.0.0.1", Process: "postgres", PID: 90},
	})
	if len(got) != 2 || got[0] != (Announced{Port: 5432, IP: "127.0.0.1", Process: "postgres", PID: 90, Source: BySocket}) ||
		got[1].IP != "0.0.0.0" || got[1].Process != "node" {
		t.Fatalf("announced %+v, want one per port, IPv4 first", got)
	}
}

func TestTheDiffOfTwoSnapshotsIsListenAndUnlisten(t *testing.T) {
	before := []Announced{{Port: 3000, IP: "0.0.0.0"}, {Port: 8081, IP: "0.0.0.0"}}
	after := []Announced{{Port: 3000, IP: "0.0.0.0"}, {Port: 5173, IP: "127.0.0.1"}}
	got := DiffAnnounced(before, after)
	if len(got) != 2 || got[0].Listen == nil || got[0].Listen.Port != 5173 || got[1].Unlisten != 8081 {
		t.Fatalf("diff %+v", got)
	}
	if kept := FilterAnnounced([]Announced{{Port: 22}, {Port: 631}, {Port: 3000}}, 1024, []int{22}); len(kept) != 1 || kept[0].Port != 3000 {
		t.Fatalf("filter %+v", kept)
	}
}

// invariant: a socket in the guest's ephemeral range is marked, never
// dropped, so an owner's share still finds it, whatever it binds; below the
// range a loopback socket is not marked, and neither is a container's
// published port.
func TestAnEphemeralListenerIsMarkedQuiet(t *testing.T) {
	got := MarkQuiet([]Announced{
		{Port: 3000, IP: "0.0.0.0", Source: BySocket}, {Port: 5173, IP: "127.0.0.1", Source: BySocket}, {Port: 8000, IP: "::1", Source: BySocket},
		{Port: 45000, IP: "0.0.0.0", Source: BySocket}, {Port: 45001, IP: "0.0.0.0", Source: ByContainer}, {Port: 45002, IP: "127.0.0.1", Source: BySocket},
	}, 32768, 60999)
	want := []bool{false, false, false, true, false, true}
	for i, l := range got {
		if l.Quiet != want[i] {
			t.Errorf("port %d is %v, want %v", l.Port, l.Quiet, want[i])
		}
	}
}
