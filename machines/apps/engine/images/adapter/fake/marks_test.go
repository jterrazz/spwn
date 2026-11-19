package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/images/contract/contracttest"
)

func TestTheFakeBuildMarksKeepTheContract(t *testing.T) {
	contracttest.BuildMarks(t, func(*testing.T) contract.BuildMarks { return NewMarks() })
}
