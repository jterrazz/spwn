package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

func TestAppliedStore(t *testing.T) {
	contracttest.AppliedStore(t, func(*testing.T) contract.AppliedStore { return fake.NewApplied() })
}
