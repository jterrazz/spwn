package calls_test

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	corestore "github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/engine"
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

// value: an engine that holds no VM and runs none; down, it does not
// answer, as while it restarts; run, when set, is called at each VM's run.
type idleEngine struct {
	down bool
	run  func()
}

func (e *idleEngine) Ping(context.Context) (string, error) {
	if e.down {
		return "", model.ErrEngineDown
	}
	return "1.0.0", nil
}
func (*idleEngine) VMs(context.Context) (map[string]machines.EngineVM, error) {
	return map[string]machines.EngineVM{}, nil
}
func (e *idleEngine) Do(_ context.Context, action, _ string) error {
	if action == engine.VMRun && e.run != nil {
		e.run()
	}
	return errNoRun
}
func (*idleEngine) EnvPorts(context.Context, string) ([]model.Reserved, error) {
	return nil, nil
}
func (*idleEngine) GuestExec(context.Context, string, string, string) (machines.GuestRan, error) {
	return machines.GuestRan{}, errNoRun
}

// value: the processes of a Mac: this one, and those that run.
type processes struct {
	mu   sync.Mutex
	self model.Process
	runs map[model.Process]bool
}

func (p *processes) Self() (model.Process, error) { return p.self, nil }

func (p *processes) Runs(q model.Process) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return q == p.self || p.runs[q]
}

type bench struct {
	m      *machines.Machinist
	store  *machinesfake.Store
	img    *images
	engine *idleEngine
	procs  *processes
	prefs  machines.Prefs
	mac    *stillMac
}

// value: a Mac of 16 GB whose clock reads 10:00 and whose engine answers,
// its image built or not.
func creating(t *testing.T, built bool, hardware ...func(context.Context) coremachine.Hardware) bench {
	t.Helper()
	hardware = append(hardware, func(context.Context) coremachine.Hardware { return coremachine.Hardware{Memory: 16 << 30, CPUs: 8} })
	w := &world.World{ThisMac: "mac"}
	w.Mac("mac").Images = []string{"station-linux"}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	store, img := machinesfake.StoreOf(w, "mac"), &images{built: built}
	e := &idleEngine{}
	procs := &processes{self: model.Process{PID: 100, Start: "1763024400.5"}, runs: map[model.Process]bool{}}
	linux := catalog.Image{Name: "station-linux", Kind: catalog.Linux, Defaults: catalog.ImageDefaults{CPUs: 4, MemoryBytes: 4 << 30, DiskBytes: 20e9}}
	mac := &stillMac{at: time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)}
	m := machines.New(mac, machinesfake.HypervisorOf(w, "mac", "hypervisor", nil), p, store, noTailnet{}, img, e, nil, machines.Options{
		ThisMachine: catalog.Mac, Images: []catalog.Image{linux}, KnownHosts: "/state/known_hosts", Logs: "/state/logs",
		Hardware: hardware[0], Processes: procs,
	})
	return bench{m: m, store: store, img: img, engine: e, procs: procs, prefs: p, mac: mac}
}

func TestAVMKeepsWhatItIsForAndWhoMadeIt(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	for _, r := range []model.CreateRequest{
		{Image: "station-linux", Name: "gate", Purpose: "Lot shell, one block per package", Creator: "studio-os-fe"},
		{Image: "station-linux", Name: "api"},
	} {
		if _, err := b.m.Create(ctx, r, nil, nil); !errors.As(err, new(*model.DidNotBootError)) {
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
		if _, err := b.m.Create(ctx, r, nil, nil); !errors.As(err, &bad) || bad.Session != session {
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

// value: the VM of that name as the fleet read says it, and the memory
// This Mac holds for it.
func readVM(t *testing.T, b bench, name string) (vm model.VM, held int64) {
	t.Helper()
	all, err := b.m.AllMachines(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := model.CapacityOf(all.Macs[0]); ok {
		for _, s := range c.VMs {
			if s.Name == name {
				held = s.Bytes
			}
		}
	}
	for _, v := range all.Macs[0].VMs {
		if v.Name == name {
			vm = v
		}
	}
	return vm, held
}

func TestACreateHoldsItsRecordWhileItsImageIsBuilt(t *testing.T) {
	ctx := context.Background()
	b := creating(t, false)
	var before, during model.VM
	var held int64
	build := func(_ context.Context, step func(string)) error {
		before, held = readVM(t, b, "web")
		step("image 2 of 3")
		during, _ = readVM(t, b, "web")
		b.img.built = true
		return nil
	}
	_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "web"}, build, nil)
	if !errors.As(err, new(*model.DidNotBootError)) {
		t.Fatalf("create: %v; want it built, cloned and kept, not booted", err)
	}
	if before.State != model.Creating || before.Birth == nil || before.Birth.Step != model.BornBuilding || held != 4<<30 {
		t.Errorf("before the build's first step web reads %s, birth %+v, %d bytes held; want Creating, building, 4 GB held", before.State, before.Birth, held)
	}
	if during.Birth == nil || during.Birth.Detail != "image 2 of 3" || during.Birth.Doing() != "image 1 of 5" {
		t.Errorf("during the build web's birth is %+v; want image 1 of 5, the build's own step its detail", during.Birth)
	}
	if kept, err := b.store.Get(ctx, "web"); err != nil || kept.Birth != nil || kept.Version != "v1" {
		t.Errorf("web's record once made = %+v, %v; want it kept, its version the image's, its birth off", kept, err)
	}
}

func TestACreateWithNoRoomIsRefusedBeforeItsImageIsBuilt(t *testing.T) {
	ctx := context.Background()
	b := creating(t, false)
	built := false
	build := func(context.Context, func(string)) error {
		built = true
		return nil
	}
	_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "big", Size: model.Resources{MemoryBytes: 60 << 30}}, build, nil)
	if !errors.As(err, new(*model.TooBigError)) || built {
		t.Fatalf("create of 60 GB on a Mac of 16: %v, image built %v; want refused for memory, nothing built", err, built)
	}
	if _, err := b.m.CheckCreate(ctx, model.CreateRequest{Image: "station-linux", Size: model.Resources{MemoryBytes: 60 << 30}}); !errors.As(err, new(*model.TooBigError)) {
		t.Errorf("its check: %v; want the memory refusal, asked before the build's question", err)
	}
	if _, err := b.store.Get(ctx, "big"); err == nil {
		t.Error("a refused create left a record")
	}
}

func TestACreateTakesTheWholeFreeMemoryAndNotAByteMore(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	free := int64(12 << 30)
	if _, err := b.m.CheckCreate(ctx, model.CreateRequest{Image: "station-linux", Size: model.Resources{MemoryBytes: free}}); err != nil {
		t.Errorf("12 GB on a Mac of 16, 12 free: %v; want it to fit", err)
	}
	_, err := b.m.CheckCreate(ctx, model.CreateRequest{Image: "station-linux", Size: model.Resources{MemoryBytes: free + 1}})
	if tooBig := new(model.TooBigError); !errors.As(err, &tooBig) || tooBig.LeftGB != 12 {
		t.Errorf("one byte over the free 12 GB: %v; want refused for memory, 12 GB left", err)
	}
}

func TestABuildThatFailsLeavesNoRecord(t *testing.T) {
	ctx := context.Background()
	b := creating(t, false)
	failing := errors.New("the recipe failed")
	_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "web"}, func(_ context.Context, step func(string)) error {
		step("image 1 of 3")
		return failing
	}, nil)
	if !errors.Is(err, failing) {
		t.Fatalf("create: %v; want the build's failure", err)
	}
	if _, err := b.store.Get(ctx, "web"); err == nil {
		t.Error("a create whose build failed left its record")
	}
}

func TestACreateInterruptedDuringItsBuildLeavesNoRecord(t *testing.T) {
	b := creating(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bound, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	building := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "web"}, func(ctx context.Context, step func(string)) error {
			step("image 1 of 3")
			close(building)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-bound.Done():
				return errors.New("never interrupted")
			}
		}, nil)
		done <- err
	}()
	select {
	case <-building:
	case <-bound.Done():
		t.Fatal("the build never began")
	}
	if vm, _ := readVM(t, b, "web"); vm.State != model.Creating {
		t.Fatalf("web reads %q while its image is built; want creating", vm.State)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("create: %v; want the interruption", err)
		}
	case <-bound.Done():
		t.Fatal("the interrupted create never ended")
	}
	if _, err := b.store.Get(context.Background(), "web"); err == nil {
		t.Error("an interrupted create left its record")
	}
}

// value: the process web's create runs in, in beingMade's bench.
var maker = model.Process{PID: 7, Start: "1763020000.25"}

// value: a bench whose store holds web being made at that step by maker,
// its last step at 09:15, three quarters of an hour before the Mac's clock;
// maker runs or not.
func beingMade(t *testing.T, step model.BirthStep, runs bool) bench {
	t.Helper()
	b := creating(t, true)
	at := time.Date(2026, 11, 13, 9, 15, 0, 0, time.UTC)
	rec := model.Record{Name: "web", Image: "station-linux", Lifetime: model.Persistent, CPUs: 4, MemoryBytes: 4 << 30, DiskBytes: 20e9, CreatedAt: at, Birth: model.BirthAt(step, true, at), Owner: &maker}
	if err := b.store.Put(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	b.procs.runs[maker] = runs
	return b
}

func TestACreateWhoseStepRunsPastHalfAnHourStillReadsCreating(t *testing.T) {
	b := beingMade(t, model.BornBuilding, true)
	if vm, held := readVM(t, b, "web"); vm.State != model.Creating || vm.Birth == nil || held != 4<<30 {
		t.Errorf("web, its build at one step for 45 min by a process that runs, reads %s %q, %d bytes held; want Creating, its 4 GB held", vm.State, vm.Detail, held)
	}
}

func TestACreateWhoseProcessRunsOutlivesItsEnginesRestart(t *testing.T) {
	for _, down := range []bool{false, true} {
		b := beingMade(t, model.BornCloning, true)
		b.engine.down = down
		vm, held := readVM(t, b, "web")
		if vm.State != model.Creating || vm.Birth == nil || held != 4<<30 {
			t.Errorf("engine down %v: web, its create's process alive, reads %s %q, %d bytes held; want Creating, its 4 GB held", down, vm.State, vm.Detail, held)
		}
		if err := b.m.CheckDelete("web"); !errors.As(err, new(*model.BeingCreatedError)) {
			t.Errorf("engine down %v: a delete of web while its create runs: %v; want it refused, being created", down, err)
		}
	}
}

func TestABirthWhoseProcessEndedReadsHalfMadeAndHoldsNoMemory(t *testing.T) {
	for _, c := range []struct {
		name string
		step model.BirthStep
		runs bool
	}{
		{"its process was killed", model.BornBuilding, false},
		{"its process was killed while its disk was cloned", model.BornCloning, false},
		{"a step this station does not know", "cloning the image", true},
	} {
		b := beingMade(t, c.step, c.runs)
		vm, held := readVM(t, b, "web")
		if vm.State != model.Stopped || vm.Detail != model.HalfMade || vm.Birth != nil || held != 0 {
			t.Errorf("%s: web reads %s %q, birth %+v, %d bytes held; want stopped, half made, no memory held", c.name, vm.State, vm.Detail, vm.Birth, held)
		}
		if err := b.m.CheckDelete("web"); err != nil {
			t.Errorf("%s: a delete of half-made web: %v; want it let through", c.name, err)
		}
	}
}

func TestAVMIsNotDeletedWhileItsCreateRunsAndAGoneRecordIsNeverMadeAgain(t *testing.T) {
	ctx := context.Background()
	b := creating(t, false)
	var refused, deleted error
	_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "web"}, func(_ context.Context, step func(string)) error {
		refused = b.m.CheckDelete("web")
		_, deleted = b.m.Delete(ctx, "web")
		if err := b.store.Delete(ctx, "web"); err != nil {
			return err
		}
		step("image 2 of 3")
		b.img.built = true
		return nil
	}, nil)
	if !errors.As(refused, new(*model.BeingCreatedError)) || !errors.As(deleted, new(*model.BeingCreatedError)) {
		t.Errorf("a delete of web while its image is built: check %v, delete %v; want both refused, being created", refused, deleted)
	}
	if failed := new(model.CreateFailedError); !errors.As(err, &failed) || failed.Step != "the record" {
		t.Errorf("create whose record went away during its build: %v; want it stopped at the record", err)
	}
	if kept, err := b.store.Get(ctx, "web"); err == nil {
		t.Errorf("a create whose record went away wrote it again: %+v", kept)
	}
}

func TestWhatIsWrittenWhileACreateBootsStaysOrIsRefused(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	var edited, kept, stopped error
	var step model.BirthStep
	b.engine.run = func() {
		if rec, err := b.store.Get(ctx, "web"); err == nil && rec.Birth != nil {
			step = rec.Birth.Step
		}
		_, edited = b.m.Edit(ctx, "web", model.Change{Resources: model.Resources{MemoryBytes: 8 << 30}})
		kept = b.m.KeepAgents("web", []string{"docs-bot"})
		stopped = b.m.StopDirect(ctx, "web")
	}
	if _, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: "web", Lifetime: model.Persistent}, nil, nil); err == nil {
		t.Fatalf("create of web: %v; want it to end at its run", err)
	}
	if step != model.BornBooting {
		t.Fatalf("the writes ran at %q; want them while web boots", step)
	}
	if !errors.As(edited, new(*model.BeingCreatedError)) {
		t.Errorf("machines edit web --memory 8G while web boots: %v; want it refused, being created", edited)
	}
	if kept != nil || stopped != nil {
		t.Fatalf("keep-agents %v, stop %v while web boots; want both done", kept, stopped)
	}
	rec, err := b.store.Get(ctx, "web")
	if err != nil {
		t.Fatal(err)
	}
	if rec.MemoryBytes != 4<<30 || rec.Want != model.WantStopped || !slices.Equal(rec.Agents, []string{"docs-bot"}) || rec.Birth != nil {
		t.Errorf("web after its create: %d bytes, want %q, agents %v, birth %+v; want its 4 GB, stopped, docs-bot kept, no birth", rec.MemoryBytes, rec.Want, rec.Agents, rec.Birth)
	}
}

func TestASaveOfAVMDeletedMeanwhileLeavesItGone(t *testing.T) {
	ctx := context.Background()
	b := creating(t, true)
	if err := b.store.Put(ctx, model.Record{Name: "web", Image: "station-linux", Lifetime: model.Persistent, MemoryBytes: 4 << 30}); err != nil {
		t.Fatal(err)
	}
	held := model.VMRecord{Name: "web", Image: "station-linux", Lifetime: model.Persistent, Want: model.WantStopped}
	if err := b.store.Delete(ctx, "web"); err != nil {
		t.Fatal(err)
	}
	if err := b.m.Save(held); !errors.As(err, new(*model.UnknownMachineError)) {
		t.Errorf("a save of web after its delete: %v; want web unknown", err)
	}
	if kept, err := b.store.Get(ctx, "web"); !errors.Is(err, corestore.ErrNotFound) {
		t.Errorf("a save of web after its delete made its record again: %+v, %v", kept, err)
	}
}

type caller struct{}

// value: a Mac of 16 GB whose every read of its hardware by one create
// waits, for up to 100 ms, until the other create has read it as many
// times: two creates that run their checks at once read their memory in
// lockstep, so neither writes its record before the other has judged it.
func lockstep() func(context.Context) coremachine.Hardware {
	var mu sync.Mutex
	reads := map[any]int{}
	return func(ctx context.Context) coremachine.Hardware {
		who := ctx.Value(caller{})
		mu.Lock()
		reads[who]++
		mine := reads[who]
		mu.Unlock()
		wait, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer stop()
		for wait.Err() == nil {
			mu.Lock()
			behind := false
			for other, n := range reads {
				behind = behind || (other != who && n < mine)
			}
			alone := len(reads) < 2
			mu.Unlock()
			if !alone && !behind {
				break
			}
			runtime.Gosched()
		}
		return coremachine.Hardware{Memory: 16 << 30, CPUs: 8}
	}
}

func TestTwoCreatesAtOnceNeverBothTakeTheMemoryLeft(t *testing.T) {
	bound, stop := context.WithTimeout(context.Background(), 60*time.Second)
	defer stop()
	for round := range 3 {
		b := creating(t, false, lockstep())
		building := make(chan string, 2)
		release := make(chan struct{})
		errs := make(chan error, 2)
		for _, name := range []string{"api", "web"} {
			go func() {
				ctx := context.WithValue(context.Background(), caller{}, name)
				_, err := b.m.Create(ctx, model.CreateRequest{Image: "station-linux", Name: name, Size: model.Resources{MemoryBytes: 8 << 30}}, func(context.Context, func(string)) error {
					building <- name
					select {
					case <-release:
					case <-bound.Done():
					}
					return errors.New("the test builds no image")
				}, nil)
				errs <- err
			}()
		}
		refused, built := 0, 0
		for refused+built < 2 {
			select {
			case err := <-errs:
				if !errors.As(err, new(*model.TooBigError)) {
					t.Fatalf("round %d: a create ended with %v before any build ended; want the memory refusal", round, err)
				}
				refused++
			case <-building:
				built++
			case <-bound.Done():
				t.Fatalf("round %d: the creates never reached their builds nor a refusal", round)
			}
		}
		close(release)
		for range built {
			select {
			case err := <-errs:
				if err == nil {
					t.Fatalf("round %d: a create whose build failed was made", round)
				}
			case <-bound.Done():
				t.Fatalf("round %d: a create never ended", round)
			}
		}
		if refused != 1 {
			t.Fatalf("round %d: %d of two creates of 8 GB were refused on a Mac with room for one; want one", round, refused)
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

func TestAVMASilentMacHoldsWithNoKindKeepsItsName(t *testing.T) {
	b := creating(t, true)
	if err := b.prefs.Macs.Set(map[string]model.MacRecord{"studio": {VMs: []model.SeenVM{{Name: "old", Lifetime: model.Persistent, CPUs: 2, MemoryBytes: 4 << 30}}}}); err != nil {
		t.Fatal(err)
	}
	_, err := b.m.CheckCreate(context.Background(), model.CreateRequest{Image: "station-linux", Name: "old"})
	if taken := new(model.NameTakenError); !errors.As(err, &taken) || !taken.VM || taken.On != "studio" {
		t.Errorf("a create named after the VM silent studio holds with no kind: %v; want the name taken, on studio", err)
	}
}

func TestTheBirthsOfACreateAreEveryStepItReachesAsMachinesListReadsThem(t *testing.T) {
	ctx := t.Context()
	b := beingMade(t, model.BornCloning, true)
	moved := make(chan struct{})
	var asked []time.Duration
	b.mac.pause = func(d time.Duration) {
		asked = append(asked, d)
		moved <- struct{}{}
		<-moved
	}
	births, ok := b.m.Births(ctx, "web")
	if !ok {
		t.Fatal("web's create runs; want its births")
	}
	bound, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()
	next := func() (model.Birth, bool) {
		select {
		case v, open := <-births:
			return v, open
		case <-bound.Done():
			t.Fatal("no birth, and the channel still open")
			return model.Birth{}, false
		}
	}
	listed := func() model.Birth {
		vm, _ := readVM(t, b, "web")
		if vm.Birth == nil {
			t.Fatal("machines.list gives web no birth")
		}
		return *vm.Birth
	}
	rewrite := func(step model.BirthStep) {
		rec, err := b.store.Get(ctx, "web")
		if err != nil {
			t.Fatal(err)
		}
		rec.Birth = model.BirthAt(step, true, rec.CreatedAt.Add(time.Hour))
		if err := b.store.Put(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}
	first, _ := next()
	<-moved
	if want := listed(); first != want {
		t.Errorf("the first birth is %+v; machines.list gives %+v", first, want)
	}
	moved <- struct{}{}
	<-moved
	rewrite(model.BornBooting)
	moved <- struct{}{}
	second, _ := next()
	<-moved
	if want := listed(); second != want || second.Step != model.BornBooting {
		t.Errorf("the next birth is %+v; machines.list gives %+v, booting", second, want)
	}
	rec, err := b.store.Get(ctx, "web")
	if err != nil {
		t.Fatal(err)
	}
	rec.Birth, rec.Owner = nil, nil
	if err := b.store.Put(ctx, rec); err != nil {
		t.Fatal(err)
	}
	moved <- struct{}{}
	if _, open := next(); open {
		t.Error("a birth after the create ended; want the channel closed")
	}
	if len(asked) != 3 || asked[0] != 2*time.Second {
		t.Errorf("polled after %v; want 2 s between reads", asked)
	}
}

func TestNoBirthsForAVMWhoseCreateDoesNotRun(t *testing.T) {
	if _, ok := beingMade(t, model.BornCloning, false).m.Births(t.Context(), "web"); ok {
		t.Error("web's create does not run; want no births")
	}
}
