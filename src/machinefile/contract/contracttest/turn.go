package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

// value: two takers of one machine's turn, built by the adapter under
// test, each as a process of that machine would hold it.

type Turns func(t *testing.T) (first, second contract.Turn)

func Turn(t *testing.T, turns Turns) {
	t.Run("a second taker is refused while the first holds the turn, then takes it", func(t *testing.T) {
		first, second := turns(t)
		release, err := first.Take(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, refused := second.Take(ctx); !errors.Is(refused, contract.ErrBusy) && !errors.Is(refused, context.Canceled) {
			t.Errorf("took a held turn: %v", refused)
		}
		release()
		again, err := second.Take(context.Background())
		if err != nil {
			t.Fatalf("a released turn is refused: %v", err)
		}
		again()
	})
}
