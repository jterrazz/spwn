package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/machines/contract/contracttest"
)

func TestTheFakeMachineStoreKeepsTheContract(t *testing.T) {
	contracttest.MachineStore(t, func(*testing.T) contract.MachineStore { return NewStore() })
}

func TestTheFakeUptimeStoreKeepsTheContract(t *testing.T) {
	contracttest.UptimeStore(t, func(*testing.T) contract.UptimeStore { return NewUptime() })
}
