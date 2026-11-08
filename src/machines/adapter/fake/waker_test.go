package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/src/machines/contract/contracttest"
)

func TestTheFakeWakerKeepsTheWakerContract(t *testing.T) {
	contracttest.Waker(t, func(*testing.T) contracttest.WakerSubject {
		k := NewWaker()
		return contracttest.WakerSubject{Waker: k, Sent: k.Sent}
	})
}
