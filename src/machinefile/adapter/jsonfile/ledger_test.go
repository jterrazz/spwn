package jsonfile_test

import (
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/jsonfile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func TestTheLedgerFileKeepsTheContract(t *testing.T) {
	contracttest.Ledger(t, func(t *testing.T) contract.Ledger {
		return jsonfile.NewLedger(local.New(nil, nil, nil, nil).FS, filepath.Join(t.TempDir(), "station", "ledger.json"))
	})
}
