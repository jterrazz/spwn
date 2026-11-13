package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/background/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/background/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestJobs(t *testing.T) {
	contracttest.Jobs(t, func(*testing.T) contracttest.Subject {
		return contracttest.Subject{Jobs: fake.New(&world.World{}, "mac-mini"), UID: 501, Dir: "/Users/owner/Library/LaunchAgents"}
	})
}
