package local

import (
	"context"
	"time"
)

type clock struct{}

func (clock) Now() time.Time { return time.Now() }

func (clock) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
