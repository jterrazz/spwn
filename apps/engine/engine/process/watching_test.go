package process

import (
	"context"
	"runtime"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func TestAMachineIsWatchedOnlyWhileASurfaceShowsItsActivity(t *testing.T) {
	d := &Engine{}
	hold := func(watch func(ctx context.Context, send func(any) error) error) (opened <-chan struct{}, leave func()) {
		ctx, cancel := context.WithCancel(t.Context())
		open, done := make(chan struct{}, 1), make(chan struct{})
		go func() {
			defer close(done)
			_ = watch(ctx, func(any) error { open <- struct{}{}; return nil })
		}()
		return open, func() { cancel(); <-done }
	}
	_, quit := hold(d.Events)
	for !d.hub.listening() {
		runtime.Gosched()
	}
	if d.Watched(engine.ThisMac) || d.Watched("linux-1") {
		t.Fatal("the menu bar, closed, watches a machine by holding its events watch")
	}
	opened, closed := hold(func(ctx context.Context, send func(any) error) error { return d.Watch(ctx, nil, send) })
	<-opened
	if !d.Watched(engine.ThisMac) || d.Watched("linux-1") {
		t.Errorf("the menu bar open: This Mac watched %v, linux-1 watched %v; want This Mac alone", d.Watched(engine.ThisMac), d.Watched("linux-1"))
	}
	shown, left := hold(func(ctx context.Context, send func(any) error) error { return d.Watch(ctx, []string{"linux-1"}, send) })
	<-shown
	closed()
	if d.Watched(engine.ThisMac) || !d.Watched("linux-1") {
		t.Errorf("the menu bar closed, linux-1's Activity shown: This Mac watched %v, linux-1 watched %v; want linux-1 alone", d.Watched(engine.ThisMac), d.Watched("linux-1"))
	}
	left()
	quit()
	if d.Watched(engine.ThisMac) || d.Watched("linux-1") {
		t.Error("a machine stays watched once every surface left")
	}
}
