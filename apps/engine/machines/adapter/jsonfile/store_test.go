package jsonfile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func disk() jsonfile.Files { return local.New(nil, nil, nil, nil).FS }

func TestMachineStore(t *testing.T) {
	contracttest.MachineStore(t, func(t *testing.T) contract.MachineStore {
		return jsonfile.New(disk(), filepath.Join(t.TempDir(), "vms"))
	})
}

func TestAPutKeepsTheV2SharesUntilTheyMoveOut(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "api.json")
	_ = os.WriteFile(at, []byte(`{"name":"api","image":"linux","lent":[{"name":"session:a","at":"2026-11-01T09:00:00Z","until":"2026-11-01T17:00:00Z"}]}`), 0o600)
	s := jsonfile.New(disk(), dir)
	r, err := s.Get(context.Background(), "api")
	if err != nil {
		t.Fatal(err)
	}
	r.Want = "stopped"
	if err := s.Put(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(at); !strings.Contains(string(data), `"until": "2026-11-01T17:00:00Z"`) {
		t.Errorf("the share was lost:\n%s", data)
	}
}

func TestAReadableListingSkipsADamagedRecordButNotANewerOne(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "api.json"), []byte(`{"name":"api","image":"linux","version":"2026.11.01"}`), 0o600)
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

func TestMigrateRewritesAnOlderRecordInTodaysLifetimeWordsAndLeavesTheRest(t *testing.T) {
	dir := t.TempDir()
	old := map[string]string{
		"api":   `{"name":"api","image":"linux","lifetime":"permanent","lent":[{"name":"session:a","at":"2026-11-01T09:00:00Z","until":"2026-11-01T17:00:00Z"}]}`,
		"v2":    `{"name":"v2","image":"linux","lifetime":"lasting"}`,
		"temp":  `{"name":"temp","image":"linux","lifetime":"throwaway"}`,
		"v2tmp": `{"name":"v2tmp","image":"linux","lifetime":"temporary"}`,
		"on":    `{"name":"on","image":"linux","lifetime":"always-on"}`,
		"today": "{\n  \"name\": \"today\",\n  \"lifetime\": \"ephemeral\",\n  \"v\": 2\n}\n",
	}
	for name, body := range old {
		_ = os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600)
	}
	s := jsonfile.New(disk(), dir)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]machines.Lifetime{"api": machines.Persistent, "v2": machines.Persistent, "temp": machines.Ephemeral, "v2tmp": machines.Ephemeral, "on": machines.AlwaysOn} {
		data, _ := os.ReadFile(filepath.Join(dir, name+".json"))
		if !strings.Contains(string(data), `"lifetime": "`+string(want)+`"`) || !strings.Contains(string(data), `"v": 2`) {
			t.Errorf("%s was not rewritten at version 2 with %s:\n%s", name, want, data)
		}
	}
	if r, err := s.Get(context.Background(), "temp"); err != nil || r.Lifetime != machines.Ephemeral {
		t.Errorf("temp reads %q, %v after the migration; want ephemeral", r.Lifetime, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "api.json")); !strings.Contains(string(data), `"until": "2026-11-01T17:00:00Z"`) {
		t.Errorf("the share was lost:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "today.json")); string(data) != old["today"] {
		t.Errorf("a record already at version 2 changed:\n%s", data)
	}
}

func TestMigrateStopsOnARecordANewerStationWrote(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "new.json"), []byte(`{"v":3,"name":"new","image":"linux"}`), 0o600)
	if err := jsonfile.New(disk(), dir).Migrate(context.Background()); err == nil {
		t.Errorf("Migrate passed a record a newer station wrote; want it stopped")
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
