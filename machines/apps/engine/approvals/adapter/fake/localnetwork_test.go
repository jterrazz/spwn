package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/approvals/contract"
	"github.com/jterrazz/studio-station/apps/engine/approvals/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeProbeKeepsTheContract(t *testing.T) {
	contracttest.LocalNetwork(t, func(_ *testing.T, answer approvals.State) contract.LocalNetwork {
		w := &world.World{ThisMac: "mac-mini"}
		w.Mac("mac-mini").LocalNetwork = answer.Hello()
		return LocalNetworkOf(w, "mac-mini")
	})
}
