//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
)

func TestInspect_ShowsDetails(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		Inspect().
		ExpectConfig("default")
}
