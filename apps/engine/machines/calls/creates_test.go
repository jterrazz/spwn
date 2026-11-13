package calls_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var errNoRun = errors.New("the test runs no VM")

// value: an image of this Mac, built or not.
type images struct{ built bool }

func (i *images) Latest(_ context.Context, image string) (disk, version string, err error) {
	if !i.built {
		return "", "", &model.NoImageError{Image: image}
	}
	return image, "v1", nil
}

func (*images) KeyPath() string { return "/home/.ssh/station" }

// value: an engine that answers and holds no VM, and runs none.
type idleEngine struct{}

func (idleEngine) Ping(context.Context) (string, error) { return "1.0.0", nil }
func (idleEngine) VMs(context.Context) (map[string]machines.EngineVM, error) {
	return map[string]machines.EngineVM{}, nil
}
func (idleEngine) Do(context.Context, string, string) error { return errNoRun }
func (idleEngine) EnvPorts(context.Context, string) ([]model.Reserved, error) {
	return nil, nil
}
func (idleEngine) GuestExec(context.Context, string, string, string) (machines.GuestRan, error) {
	return machines.GuestRan{}, errNoRun
}

type bench struct {
	m     *machines.Machinist
	store *machinesfake.Store
	img   *images
}

// value: a Mac of 16 GB whose engine runs, its image built or not.
func creating(t *testing.T, built bool) bench {
	t.Helper()
	w := &world.World{ThisMac: "mac"}
	w.Mac("mac").Images = []string{"station-linux"}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	store, img := machinesfake.StoreOf(w, "mac"), &images{built: built}
	linux := catalog.Image{Name: "station-linux", Kind: catalog.Linux, Defaults: catalog.ImageDefaults{CPUs: 4, MemoryBytes: 4 << 30, DiskBytes: 20e9}}
	m := machines.New(&stillMac{at: time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)}, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, store, noTailnet{}, img, idleEngine{}, nil, machines.Options{
		ThisMachine: catalog.Mac, Images: []catalog.Image{linux}, KnownHosts: "/state/known_hosts", Logs: "/state/logs",
		Hardware: func(context.Context) coremachine.Hardware { return coremachine.Hardware{Memory: 16 << 30, CPUs: 8} },
	})
	return bench{m: m, store: store, img: img}
}

func TestAVMKeepsWhatItIsForAndWhoMadeIt(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	for _, r := range []model.CreateRequest{
		{Image: "station-linux", Name: "gate", Purpose: "Lot shell, one block per package", Creator: "studio-os-fe"},
		{Image: "station-linux", Name: "api"},
	} {
		if _, err := b.m.Create(ctx, r, nil); !errors.As(err, new(*model.DidNotBootError)) {
			t.Fatalf("create %s: %v; want it kept, not booted", r.Name, err)
		}
	}
	kept, err := b.store.Get(ctx, "gate")
	if err != nil || kept.Purpose != "Lot shell, one block per package" || kept.Creator != "studio-os-fe" {
		t.Fatalf("gate's record = %+v, %v; want its purpose and its session", kept, err)
	}
	all, err := b.m.AllMachines(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	read := map[string]model.VM{}
	for _, vm := range all.Macs[0].VMs {
		read[vm.Name] = vm
	}
	if gate := read["gate"]; gate.Purpose != "Lot shell, one block per package" || gate.Creator != "studio-os-fe" {
		t.Errorf("gate reads for %q by %q; want its purpose by studio-os-fe", gate.Purpose, gate.Creator)
	}
	if api := read["api"]; api.Purpose != "" || api.Creator != model.ByOwner {
		t.Errorf("api reads for %q by %q; want no purpose, by you", api.Purpose, api.Creator)
	}
}

func TestASessionNameThatIsNoNameOrReadsAsTheOwnerIsRefused(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	for _, session := range []string{model.ByOwner, "Lot_Shell", "studio os"} {
		r := model.CreateRequest{Image: "station-linux", Name: "gate", Creator: session}
		var bad *model.BadCreatorError
		if _, err := b.m.Create(ctx, r, nil); !errors.As(err, &bad) || bad.Session != session {
			t.Errorf("create for the session %q: %v; want it refused for its session's name", session, err)
		}
		if _, err := b.m.CheckCreate(ctx, r); !errors.As(err, new(*model.BadCreatorError)) {
			t.Errorf("its check for %q: %v; want the same refusal", session, err)
		}
		if _, err := b.store.Get(ctx, "gate"); err == nil {
			t.Fatalf("a create for the session %q left a record", session)
		}
	}
}

func TestAPurposeIsOneLineOfAtMostItsLimit(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	for _, c := range []struct {
		purpose string
		ok      bool
	}{
		{strings.Repeat("é", model.PurposeMax), true},
		{"Lot shell, one block per package", true},
		{strings.Repeat("a", model.PurposeMax+1), false},
		{"one\ntwo", false},
		{"one\ttwo", false},
		{"one\u2028two", false},
		{"one\u2029two", false},
		{"Note\u00a0: a lot", true},
		{"Lot 1\u202f000", true},
		{"Shell \U0001F468\u200d\U0001F4BB", true},
		{"bad \xff byte", false},
	} {
		_, err := b.m.CheckCreate(ctx, model.CreateRequest{Image: "station-linux", Purpose: c.purpose})
		if refused := errors.As(err, new(*model.BadPurposeError)); refused == c.ok {
			t.Errorf("purpose %q: %v; want refused %v", c.purpose, err, !c.ok)
		}
	}
}
