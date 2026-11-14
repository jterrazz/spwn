package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

type base = Shares

type touched struct {
	base
	calls int
}

func (s *touched) CheckFrom(context.Context, string, string, string) error {
	s.calls++
	return nil
}

func (s *touched) Share(context.Context, string, string, time.Duration) (accounts.Shared, error) {
	s.calls++
	return accounts.Shared{}, nil
}

func TestAShareOutsideItsSpanIsRefusedBeforeAnyEffect(t *testing.T) {
	for _, term := range []time.Duration{31 * 365 * 24 * time.Hour, -time.Hour, accounts.LongestShare + time.Second, time.Second} {
		held := &touched{}
		_, err := Share.Run(context.Background(), Deps{Shares: held}, ShareIn{Account: "github", With: "linux-1", For: term}, nil)
		var failed *fail.Error
		if !errors.As(err, &failed) || failed.Code != TermOutOfBounds {
			t.Fatalf("a share for %s ended with %v; want %s", term, err, TermOutOfBounds)
		}
		if held.calls != 0 {
			t.Fatalf("a share for %s reached the engine %d times before being refused", term, held.calls)
		}
	}
}
