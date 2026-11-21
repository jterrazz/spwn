package local

import (
	"context"
	"time"
)

type clock struct{}

func (clock) Now() time.Time { return time.Now() }

func (c clock) Sleep(ctx context.Context, d time.Duration) error {
	return c.Await(ctx, nil, d)
}

func (clock) Await(ctx context.Context, ready <-chan struct{}, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ready:
		return nil
	case <-t.C:
		return nil
	}
}
