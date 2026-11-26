package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract/contracttest"
)

func TestMinds(t *testing.T) {
	contracttest.Minds(t, func(t *testing.T) (contract.Minds, string) { return fake.New(), t.TempDir() })
}
