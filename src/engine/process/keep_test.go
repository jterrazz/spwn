package process

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/engine"
)

func TestTheEventsWatchStreamsEachEventOfTheClosedListOnceAndNoOtherWord(t *testing.T) {
	d := &Engine{o: Options{Join: func(parts ...string) string { return strings.Join(parts, " / ") }}}
	ctx, cancel := context.WithCancel(t.Context())
	got := make(chan engine.Event, 4)
	opened := make(chan engine.Opened, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = d.Events(ctx, func(v any) error {
			if o, ok := v.(engine.Opened); ok {
				opened <- o
				return nil
			}
			ev, _ := v.(engine.Event)
			got <- ev
			return nil
		})
	}()
	if o := <-opened; o.Stream != engine.EventsStream {
		t.Errorf("the events watch opened as %q, want %q", o.Stream, engine.EventsStream)
	}
	for !d.hub.listening() {
		runtime.Gosched()
	}
	d.tell(ctx, "infra", engine.Back("infra"))
	d.Tell("jterrazz-macbook-pro", "memory", "This Mac: Memory pressure critical")
	d.Tell("jterrazz-macbook-pro", activity.SubjectDisk, "This Mac: 3 % of its disk is free")
	d.tell(ctx, "infra", engine.PoweredOff("infra"))
	for _, want := range []engine.Notice{
		{Kind: engine.EventBack, Icon: "checkmark.circle.fill", Text: "infra is back / all 4 health steps pass"},
		{Kind: engine.EventDiskLow, Icon: "exclamationmark.circle.fill", Text: "This Mac: 3 % of its disk is free"},
		{Kind: engine.EventPoweredOff, Icon: "xmark.octagon.fill", Text: "infra powered off from inside / it stays stopped"},
	} {
		if ev := <-got; ev.Event != "notice" || *ev.Notice != want {
			t.Errorf("streamed %+v, want the notice %+v", ev, want)
		}
	}
	cancel()
	<-done
	select {
	case ev := <-got:
		t.Errorf("streamed %+v once more", ev)
	default:
	}
}
