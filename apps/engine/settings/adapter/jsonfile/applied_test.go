package jsonfile_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

func TestAppliedStore(t *testing.T) {
	contracttest.AppliedStore(t, func(t *testing.T) contract.AppliedStore {
		return jsonfile.NewApplied(local.New(nil, nil, nil, nil).FS, t.TempDir())
	})
}
