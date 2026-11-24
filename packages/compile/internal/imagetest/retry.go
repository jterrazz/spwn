package imagetest

import (
	"context"
	"time"
)

const buildAttempts = 3

func buildBackoff(n int) time.Duration { return time.Duration(n) * 10 * time.Second }

func retry(ctx context.Context, attempts int, backoff func(int) time.Duration, attempt func() error) error {
	var err error
	for n := 1; n <= attempts; n++ {
		if err = attempt(); err == nil {
			return nil
		}
		if n == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(backoff(n)):
		}
	}
	return err
}
