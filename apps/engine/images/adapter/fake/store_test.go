package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/images/contract/contracttest"
)

func TestTheFakeImageStoreKeepsTheContract(t *testing.T) {
	contracttest.ImageStore(t, func(*testing.T) contract.ImageStore { return NewStore() })
}
