package jsonfile_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the machine's files with the lock wire gives the ledger.
type lockable struct{ local.FS }

func (lockable) Lock(path string) (func(), error) { return local.Exclusive(context.Background(), path) }

func open(at string) *jsonfile.Ledger {
	return jsonfile.NewLedger(lockable{local.New(nil, nil, nil, nil).FS}, at)
}

func TestTheLedgerFileKeepsTheContract(t *testing.T) {
	contracttest.Ledger(t, func(t *testing.T) contract.Ledger {
		return open(filepath.Join(t.TempDir(), "station", "ledger.json"))
	})
}

func TestTwoAppliesAtOnceLoseNoRecord(t *testing.T) {
	// invariant: two ledgers over one file, as two applies at once hold,
	// keep every record either puts, and leave no temporary file behind.
	at := filepath.Join(t.TempDir(), "station", "ledger.json")
	var wg sync.WaitGroup
	for n, l := range []*jsonfile.Ledger{open(at), open(at)} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 40 {
				pkg := "app-" + strconv.Itoa(n) + "-" + strconv.Itoa(i)
				if err := l.Put(context.Background(), machinefile.Installed{Package: pkg, Method: "first", Machinefile: "laptop"}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	got, err := open(at).List(context.Background())
	if err != nil || len(got) != 80 {
		t.Fatalf("List = %d records, %v; want 80", len(got), err)
	}
	if _, err := local.New(nil, nil, nil, nil).FS.Stat(at + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a temporary file stays: %v", err)
	}
}

const ledgerV1 = `{
  "v": 1,
  "installed": [{"item": "jq", "method": "first", "machinefile": "laptop", "at": "2026-11-01T09:00:00Z"}],
  "blocks": [{"id": "path", "item": "jq", "digest": "sha256:ab", "at": "2026-11-01T09:00:00Z"}],
  "night": {"at": "2026-11-02T03:00:00Z", "machinefile": "laptop"}
}
`

func v1Ledger(t *testing.T) string {
	t.Helper()
	at := filepath.Join(t.TempDir(), "station", "ledger.json")
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(ledgerV1), 0o600); err != nil {
		t.Fatal(err)
	}
	return at
}

func TestALedgerAtVersion1IsRefusedUntilMigrateKeepsItsPackages(t *testing.T) {
	at := v1Ledger(t)
	l := open(at)
	var older *store.OlderError
	if _, err := l.List(context.Background()); !errors.As(err, &older) || older.Version != 1 || older.Current != 2 {
		t.Fatalf("List of a version 1 ledger = %v; want it refused as older, until station init", err)
	}
	if err := l.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	installed, err := l.List(context.Background())
	if err != nil || len(installed) != 1 || installed[0].Package != "jq" || installed[0].Method != "first" {
		t.Errorf("List after Migrate = %+v, %v; want jq by its first method", installed, err)
	}
	blocks, err := l.Blocks(context.Background())
	if err != nil || len(blocks) != 1 || blocks[0].Package != "jq" || blocks[0].ID != "path" {
		t.Errorf("Blocks after Migrate = %+v, %v; want jq's path block", blocks, err)
	}
	if night, ok, err := l.Night(context.Background()); err != nil || !ok || night.Machinefile != "laptop" {
		t.Errorf("Night after Migrate = %+v, %v, %v; want laptop's", night, ok, err)
	}
	data, _ := os.ReadFile(at)
	if !strings.Contains(string(data), `"v": 2`) || strings.Contains(string(data), `"item"`) {
		t.Errorf("the ledger is not written at version 2:\n%s", data)
	}
	if err := l.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(at); !bytes.Equal(again, data) {
		t.Errorf("a second Migrate changed a ledger at version 2:\n%s", again)
	}
}

// value: the machine's files, whose first lock lets another migration run
// to its end before it is taken, as a second station init would.
type overtaken struct {
	lockable
	first func()
}

func (o *overtaken) Lock(path string) (func(), error) {
	if o.first != nil {
		run := o.first
		o.first = nil
		run()
	}
	return o.lockable.Lock(path)
}

func TestAMigrateOvertakenByAnotherReadsTheLedgerAgainUnderItsLock(t *testing.T) {
	at := v1Ledger(t)
	files := &overtaken{lockable: lockable{local.New(nil, nil, nil, nil).FS}}
	files.first = func() {
		if err := open(at).Migrate(context.Background()); err != nil {
			t.Error(err)
		}
	}
	if err := jsonfile.NewLedger(files, at).Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	installed, err := open(at).List(context.Background())
	if err != nil || len(installed) != 1 || installed[0].Package != "jq" {
		t.Errorf("List after two migrations = %+v, %v; want jq kept", installed, err)
	}
}
