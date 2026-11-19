package ports_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func TestAReservationIsStickyAndTakesTheLowestFreePort(t *testing.T) {
	p := ports.Policy{Low: 40000, High: 40002, MaxPerVM: 2}
	held := []ports.Reservation{{VM: "api", GuestPort: 8081, MacPort: 40000}}
	if r, err := ports.Reserve(p, held, "api", 8081, nil); err != nil || r.MacPort != 40000 {
		t.Fatalf("a reservation moved: %+v %v", r, err)
	}
	busy := func(port int) bool { return port == 40001 }
	if r, err := ports.Reserve(p, held, "web", 3000, busy); err != nil || r.MacPort != 40002 {
		t.Fatalf("reserved %+v %v, want 40002 past the port busy on the Mac", r, err)
	}
}

func TestAVMHoldsAtMostItsShareAndTheRangeEnds(t *testing.T) {
	held := []ports.Reservation{{VM: "api", GuestPort: 8081, MacPort: 40000}, {VM: "api", GuestPort: 8082, MacPort: 40001}}
	two := ports.Policy{Low: 40000, High: 40009, MaxPerVM: 2}
	var full *ports.VMFull
	if _, err := ports.Reserve(two, held, "api", 3000, nil); !errors.As(err, &full) || err.Error() != "api has 2 shared ports, the most a VM may" {
		t.Fatalf("err %v, want api full", err)
	}
	if r, err := ports.Reserve(two, held, "api", 8081, nil); err != nil || r.MacPort != 40000 {
		t.Fatalf("a full VM lost the port it holds: %+v %v", r, err)
	}
	var taken *ports.RangeFull
	if _, err := ports.Reserve(ports.Policy{Low: 40000, High: 40000, MaxPerVM: 9}, held, "web", 3000, nil); !errors.As(err, &taken) {
		t.Fatalf("err %v, want the range taken", err)
	}
}
