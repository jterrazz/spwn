package each_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/remote/each"
)

func TestEveryMacAnswersAsItComesAndAFastOneIsNotKeptWaiting(t *testing.T) {
	answered := make(chan struct{})
	results := each.Do(t.Context(), []string{"mac-mini", "studio"}, func(_ context.Context, mac string) (string, error) {
		if mac == "mac-mini" {
			<-answered
		}
		return mac, nil
	})
	first := <-results
	close(answered)
	second := <-results
	if first.Value != "studio" || second.Value != "mac-mini" {
		t.Errorf("the answers came %q then %q, want the studio first while the mini still thinks", first.Value, second.Value)
	}
}

func TestAMiniAnsweringAfterEightSecondsIsCutAtFiveAndTheStudioIsNotKeptWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		before, cancelBefore := context.WithTimeout(t.Context(), 5*time.Second-time.Millisecond)
		defer cancelBefore()
		after, cancelAfter := context.WithTimeout(t.Context(), 5*time.Second+time.Millisecond)
		defer cancelAfter()
		for r := range each.Do(t.Context(), []string{"mac-mini", "studio"}, func(ctx context.Context, mac string) (string, error) {
			if mac == "mac-mini" {
				eight, cancel := context.WithTimeout(t.Context(), 8*time.Second)
				defer cancel()
				select {
				case <-eight.Done():
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			return mac, nil
		}) {
			switch {
			case r.Mac == "studio" && before.Err() != nil:
				t.Errorf("the studio's answer was kept waiting past 5s")
			case r.Mac == "mac-mini" && !errors.Is(r.Err, context.DeadlineExceeded):
				t.Errorf("the mini that answers after 8s: %v, want out of its budget", r.Err)
			case r.Mac == "mac-mini" && (before.Err() == nil || after.Err() != nil):
				t.Errorf("the mini was cut before 5s or after it, want at 5s")
			}
		}
	})
}

func TestAtMostFourMacsAreAskedAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var asking atomic.Int32
		release := make(chan struct{})
		done := make(chan struct{})
		go func() {
			each.All(t.Context(), []string{"a", "b", "c", "d", "e", "f", "g"}, func(_ context.Context, mac string) (string, error) {
				asking.Add(1)
				<-release
				asking.Add(-1)
				return mac, nil
			})
			close(done)
		}()
		synctest.Wait()
		if n := asking.Load(); n != 4 {
			t.Errorf("%d of 7 Macs were asked at once, want 4", n)
		}
		close(release)
		<-done
	})
}
