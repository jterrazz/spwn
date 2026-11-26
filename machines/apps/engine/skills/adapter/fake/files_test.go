package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/skills/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract/contracttest"
)

func TestFiles(t *testing.T) {
	contracttest.Files(t, func(t *testing.T) (contract.Files, string) { return fake.New(), t.TempDir() })
}
