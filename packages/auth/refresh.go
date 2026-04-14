package auth

import (
	"context"
	"log"
	"time"
)

// StartRefresh launches a background goroutine that calls SyncCredentials
// at the given interval. Returns a cancel function to stop the refresh.
func StartRefresh(ctx context.Context, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := SyncCredentials(); err != nil {
					log.Printf("warning: credential refresh failed: %v", err)
				}
			}
		}
	}()
	return cancel
}
