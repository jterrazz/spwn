package files_test

import (
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/local"
)

func TestTheTurnIsOneFileLockOfTheMachine(t *testing.T) {
	contracttest.Turn(t, func(t *testing.T) (contract.Turn, contract.Turn) {
		at := filepath.Join(t.TempDir(), "station", "runs.lock")
		return files.NewTurn(at, local.ExclusiveWithin), files.NewTurn(at, local.ExclusiveWithin)
	})
}
