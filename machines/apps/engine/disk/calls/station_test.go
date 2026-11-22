package calls

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
)

// value: a hypervisor whose disks the test changes between two reads.
type changingStation struct {
	mu    sync.Mutex
	held  *disk.Held
	err   error
	freed []string
	// value: while set, the hypervisor answers nothing until it is closed.
	asked chan struct{}
}

func (c *changingStation) Held(ctx context.Context) (*disk.Held, error) {
	c.mu.Lock()
	asked := c.asked
	c.mu.Unlock()
	if asked != nil {
		select {
		case <-asked:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.held, c.err
}

func (c *changingStation) Free(_ context.Context, frees []disk.Freeable) []disk.Outcome {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]disk.Outcome, 0, len(frees))
	for _, f := range frees {
		c.freed = append(c.freed, f.Name)
		out = append(out, disk.Outcome{Name: f.Name, Result: catalog.StateOK, Freed: f.Bytes})
	}
	return out
}

func (c *changingStation) set(held *disk.Held, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.held, c.err = held, err
}

func readings(t *testing.T, s *Surveyor) []disk.Reading {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates, err := Read.Run(ctx, Deps{Disk: s, Name: "mbp"}, ReadIn{}, nil)
	if err != nil {
		t.Fatalf("disk.read: %v", err)
	}
	var out []disk.Reading
	for u := range updates {
		out = append(out, u.Value)
	}
	if ctx.Err() != nil {
		t.Fatal("disk.read never ended")
	}
	return out
}

func firstReading(t *testing.T, s *Surveyor) disk.Reading {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates, err := Read.Run(ctx, Deps{Disk: s, Name: "mbp"}, ReadIn{}, nil)
	if err != nil {
		t.Fatalf("disk.read: %v", err)
	}
	u, ok := <-updates
	if !ok {
		t.Fatal("disk.read sent no reading")
	}
	return u.Value
}

func last(t *testing.T, s *Surveyor) disk.Reading {
	t.Helper()
	all := readings(t, s)
	if len(all) == 0 {
		t.Fatal("disk.read sent no reading")
	}
	return all[len(all)-1]
}

func kindSet(r disk.Reading) map[disk.Kind]int64 {
	out := map[disk.Kind]int64{}
	for _, a := range r.Kinds {
		out[a.Kind] = a.Bytes
	}
	return out
}

// value: a Mac's volume whose APFS container's other volumes hold 100 bytes.
type withSystem struct{ *fake.Usage }

func (withSystem) System(context.Context) int64 { return 100 }

func onMac(station Station) *Surveyor {
	usage := fake.NewUsage(map[string]int64{"/h/quick/cache/f": 40}, 1000, 500)
	apps := []catalog.App{{Name: "quick", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "quick"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/quick/*", Level: catalog.LevelCache}}}}}
	return NewSurveyor(onMachine{Usage: usage}, apps, SurveyorOptions{Disk: withSystem{usage}, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}, Station: station})
}

func TestEachReadSeesTheMachinesAndImagesOfItsMoment(t *testing.T) {
	station := &changingStation{held: &disk.Held{Machines: []disk.MachineDisk{{Name: "api", Bytes: 50}}}}
	s := onMac(station)
	if got := last(t, s); got.Station == nil || len(got.Station.Machines) != 2 {
		t.Fatalf("the first read's part is %+v, want the Mac and api", got.Station)
	}
	station.set(&disk.Held{Machines: []disk.MachineDisk{{Name: "api", Bytes: 50}, {Name: "linux-1", Bytes: 70}}}, nil)
	got := last(t, s)
	if got.Station == nil || len(got.Station.Machines) != 3 || kindSet(got)[disk.KindMachineDisks] != 120 {
		t.Fatalf("the second read's part is %+v, want linux-1 made since the first read", got.Station)
	}
}

func TestAMacWithNoHypervisorHasNoPartAndNoMachines(t *testing.T) {
	got := last(t, onMac(&changingStation{}))
	if got.Station != nil {
		t.Fatalf("station's part is %+v, want none, never zeros", got.Station)
	}
	kinds := kindSet(got)
	if _, ok := kinds[disk.KindMachineDisks]; ok || kinds[disk.KindYourData] != 360 {
		t.Fatalf("kinds %+v, want no machine disks, and your data the 500 used less 100 system and 40 caches", got.Kinds)
	}
}

func TestAFailedHypervisorLeavesMachinesAndYourDataUnknown(t *testing.T) {
	got := last(t, onMac(&changingStation{err: errors.New("the hypervisor could not list its disks")}))
	kinds := kindSet(got)
	_, machines := kinds[disk.KindMachineDisks]
	_, yours := kinds[disk.KindYourData]
	if got.Station != nil || machines || yours || got.Pending {
		t.Fatalf("a failed hypervisor reads %+v, station %+v: want no part, no machines, no your data, nothing pending", got.Kinds, got.Station)
	}
}

func TestARereadShowsTheMachinesTheLastReadHeardWhileTheHypervisorIsAskedAgain(t *testing.T) {
	station := &changingStation{held: &disk.Held{Machines: []disk.MachineDisk{{Name: "api", Bytes: 50}}}}
	s := onMac(station)
	last(t, s)
	asked := make(chan struct{})
	defer close(asked)
	station.mu.Lock()
	station.asked = asked
	station.mu.Unlock()
	got := firstReading(t, s)
	kinds := kindSet(got)
	_, yours := kinds[disk.KindYourData]
	if got.Station == nil || kinds[disk.KindMachineDisks] != 50 || !yours || !got.Pending {
		t.Fatalf("a re-read's first reading is %+v, station %+v: want api's 50 and your data the last read heard, pending while the hypervisor is asked", got.Kinds, got.Station)
	}
}

func TestARereadTheHypervisorFailsKeepsTheMachinesTheLastReadHeard(t *testing.T) {
	station := &changingStation{held: &disk.Held{Machines: []disk.MachineDisk{{Name: "api", Bytes: 50}}}}
	s := onMac(station)
	last(t, s)
	station.set(nil, errors.New("the hypervisor could not list its disks"))
	got := last(t, s)
	kinds := kindSet(got)
	_, yours := kinds[disk.KindYourData]
	if got.Station == nil || kinds[disk.KindMachineDisks] != 50 || !yours || got.Pending {
		t.Fatalf("a re-read the hypervisor fails reads %+v, station %+v: want api's 50 and your data the last read heard, nothing pending", got.Kinds, got.Station)
	}
}
