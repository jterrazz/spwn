package jsonfile_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

func disk() jsonfile.Files { return local.New(nil, nil, nil, nil).FS }

func TestMachineStore(t *testing.T) {
	contracttest.MachineStore(t, func(t *testing.T) contract.MachineStore {
		return jsonfile.New(disk(), filepath.Join(t.TempDir(), "vms"))
	})
}

func TestARecordAnOlderStationWroteIsRefused(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "api.json")
	old := `{"name":"api","image":"linux","lifetime":"permanent"}`
	_ = os.WriteFile(at, []byte(old), 0o600)
	var older *store.OlderError
	if _, err := jsonfile.New(disk(), dir).Get(context.Background(), "api"); !errors.As(err, &older) || older.Version != 1 || older.Current != 2 {
		t.Fatalf("Get of a version 1 record = %v; want it refused as older", err)
	}
	if data, _ := os.ReadFile(at); string(data) != old {
		t.Errorf("the refused record was rewritten:\n%s", data)
	}
}

func TestAReadableListingSkipsADamagedRecordButNotANewerOne(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "api.json"), []byte(`{"v":2,"name":"api","image":"linux","version":"2026.11.01"}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "x.json"), []byte(`{"name": "x", "ima`), 0o600)
	s := jsonfile.New(disk(), dir)
	if _, err := s.List(context.Background()); err == nil {
		t.Errorf("List passed a damaged record; want it named")
	}
	list, err := s.ListReadable(context.Background())
	if err != nil || len(list) != 1 || list[0].Name != "api" {
		t.Fatalf("ListReadable = %+v, %v; want api alone", list, err)
	}
	_ = os.WriteFile(filepath.Join(dir, "new.json"), []byte(`{"v":3,"name":"new","image":"linux"}`), 0o600)
	if _, err := s.ListReadable(context.Background()); err == nil {
		t.Errorf("ListReadable passed a record a newer station wrote; want it refused")
	}
}

// value: the disk, whose next read runs onRead once it has read, and whose lock is one every store over it shares, as flock is
// every process's: it says on waiting when a second taker waits for it.
type racingDisk struct {
	jsonfile.Files
	onRead  func()
	held    sync.Mutex
	waiting chan struct{}
	taken   bool
	mu      sync.Mutex
}

func (d *racingDisk) ReadFile(p string) ([]byte, error) {
	data, err := d.Files.ReadFile(p)
	d.mu.Lock()
	hook := d.onRead
	d.onRead = nil
	d.mu.Unlock()
	if hook != nil {
		hook()
	}
	return data, err
}

func (d *racingDisk) Hold(context.Context, string) (func(), error) {
	d.mu.Lock()
	if d.taken {
		close(d.waiting)
	}
	d.taken = true
	d.mu.Unlock()
	d.held.Lock()
	return d.held.Unlock, nil
}

// invariant: two amends of one record never interleave, from two
// processes (two stores over one folder, one lock): the engine's save
// that read the create's birth while the create ended it never writes
// that birth back, so none outlives its create.
func TestTwoAmendsOfARecordNeverInterleave(t *testing.T) {
	ctx := context.Background()
	racing := &racingDisk{Files: disk(), waiting: make(chan struct{})}
	dir := filepath.Join(t.TempDir(), "vms")
	create := jsonfile.New(racing, dir)
	booting := &machines.Birth{Progress: 400, Step: machines.BornBooting, At: time.Unix(0, 0).UTC()}
	if err := create.Put(ctx, machines.Record{Name: "linux-4", Birth: booting}); err != nil {
		t.Fatal(err)
	}
	ended := make(chan error, 1)
	hook := func() {
		go func() {
			ended <- create.Amend(ctx, "linux-4", func(r *machines.Record) { r.Birth = nil })
		}()
		select {
		case <-racing.waiting:
		case err := <-ended:
			ended <- err
		}
	}
	racing.mu.Lock()
	racing.onRead = hook
	racing.mu.Unlock()
	engine := jsonfile.New(racing, dir)
	if err := engine.Amend(ctx, "linux-4", func(r *machines.Record) { r.Env = map[string]string{"PORT": "40001"} }); err != nil {
		t.Fatal(err)
	}
	if err := <-ended; err != nil {
		t.Fatal(err)
	}
	got, err := create.Get(ctx, "linux-4")
	if err != nil || got.Birth != nil || got.Env["PORT"] != "40001" {
		t.Errorf("record after the save and the create's end = %+v, %v; want no birth, PORT 40001", got, err)
	}
}

// value: the disk with its flock, as the machine wires the records.
type flocked struct{ jsonfile.Files }

func (flocked) Hold(ctx context.Context, path string) (func(), error) {
	return local.Exclusive(ctx, path)
}

// invariant: a claim holds the records against a claim of another store
// over the same folder, as of another process, from its judge's read to its
// write; the one that waits for it gives up when its ctx ends, its judge
// never run.
func TestAClaimHoldsTheRecordsAgainstAnotherProcessAndItsWaitEndsWithItsContext(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "vms")
	first, second := jsonfile.New(flocked{disk()}, dir), jsonfile.New(flocked{disk()}, dir)
	judging, release := make(chan struct{}), make(chan struct{})
	bound, stop := context.WithTimeout(ctx, 20*time.Second)
	defer stop()
	held := make(chan error, 1)
	go func() {
		held <- first.Claim(ctx, func([]machines.Record) (machines.Record, error) {
			close(judging)
			select {
			case <-release:
			case <-bound.Done():
			}
			return machines.Record{Name: "api"}, nil
		})
	}()
	select {
	case <-judging:
	case <-bound.Done():
		t.Fatal("the first claim never judged")
	}
	short, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	judged := false
	err := second.Claim(short, func([]machines.Record) (machines.Record, error) {
		judged = true
		return machines.Record{Name: "web"}, nil
	})
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) || judged {
		t.Errorf("a claim while another store's claim judges: %v, judged %v; want it to wait, then end with its context, never judged", err, judged)
	}
	if err := <-held; err != nil {
		t.Fatal(err)
	}
	if _, err := first.Get(ctx, "web"); err == nil {
		t.Error("the claim that gave up wrote its record")
	}
}

type openedRecords struct {
	jsonfile.Files
	reads, listings int
}

func (o *openedRecords) ReadFile(p string) ([]byte, error) {
	o.reads++
	return o.Files.ReadFile(p)
}

func (o *openedRecords) ReadDir(p string) ([]fs.DirEntry, error) {
	o.listings++
	return o.Files.ReadDir(p)
}

func TestTheRecordsAreListedAgainOnlyWhenOneChanged(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "vms")
	files := &openedRecords{Files: disk()}
	s := jsonfile.New(files, dir)
	for _, name := range []string{"api", "web"} {
		if err := s.Put(ctx, machines.Record{Name: name, Image: "linux", CPUs: 2, Env: map[string]string{"A": "1"}}); err != nil {
			t.Fatal(err)
		}
	}
	names := func() string {
		got, err := s.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(got))
		for _, r := range got {
			out = append(out, r.Name)
		}
		return strings.Join(out, ",")
	}

	if names() != "api,web" {
		t.Fatalf("listed %q", names())
	}
	reads, listings := files.reads, files.listings
	for range 5 {
		names()
	}
	if files.reads != reads || files.listings != listings {
		t.Errorf("5 lists of unchanged records opened %d files and %d folders; want none", files.reads-reads, files.listings-listings)
	}

	got, _ := s.List(ctx)
	got[0].Env["A"] = "changed"
	if again, _ := s.List(ctx); again[0].Env["A"] != "1" {
		t.Errorf("a caller changing a listed record changed the next listing: %v", again[0].Env)
	}

	other := jsonfile.New(disk(), dir)
	_ = other.Put(ctx, machines.Record{Name: "db", Image: "linux"})
	if names() != "api,db,web" {
		t.Errorf("after another store added a record, listed %q; want api,db,web", names())
	}
	_ = os.WriteFile(filepath.Join(dir, "web.json"), []byte(`{"v":2,"name":"web","image":"linux","cpus":8}`), 0o600)
	if got, _ := s.List(ctx); got[2].CPUs != 8 {
		t.Errorf("after a record was rewritten by hand, web has %d CPUs; want 8", got[2].CPUs)
	}
	_ = other.Delete(ctx, "db")
	if names() != "api,web" {
		t.Errorf("after a record was deleted, listed %q; want api,web", names())
	}
}

func TestARecordWrittenByAnotherStoreOfTheProcessIsListedWhateverTheDisksStamp(t *testing.T) {
	ctx := context.Background()
	files := localtest.NewFS()
	mine, theirs := jsonfile.New(files, "/vms"), jsonfile.New(files, "/vms")
	for _, cpus := range []int{2, 4, 8} {
		if err := theirs.Put(ctx, machines.Record{Name: "api", Image: "linux", CPUs: cpus}); err != nil {
			t.Fatal(err)
		}
		got, err := mine.List(ctx)
		if err != nil || len(got) != 1 || got[0].CPUs != cpus {
			t.Fatalf("after another store wrote %d CPUs, listed %+v (%v)", cpus, got, err)
		}
	}
}
