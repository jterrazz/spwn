package jsonfile_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

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
