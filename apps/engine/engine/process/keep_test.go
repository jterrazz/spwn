package process

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/engine"
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
	added := []engine.Notice{
		{Kind: engine.NoticeAgentWaiting, Icon: "bubble.left", Text: "review-bot is waiting for you"},
		{Kind: engine.NoticeAgentFinished, Icon: "checkmark.circle.fill", Text: "PR #42 merged / review-bot finished"},
		{Kind: engine.NoticeVMReady, Icon: "checkmark.circle.fill", Text: "linux-4 is ready on mac-mini"},
		{Kind: engine.NoticeMacBack, Icon: "checkmark.circle.fill", Text: "mac-mini is back"},
		{Kind: engine.NoticeShareEnded, Icon: "stop.fill", Text: "linux-1:5173 is no longer shared"},
	}
	for _, want := range []engine.Notice{
		{Kind: engine.EventBack, Icon: "checkmark.circle.fill", Text: "infra is back / all 4 health steps pass"},
		{Kind: engine.EventDiskLow, Icon: "exclamationmark.circle.fill", Text: "This Mac: 3 % of its disk is free"},
		{Kind: engine.EventPoweredOff, Icon: "xmark.octagon.fill", Text: "infra powered off from inside / it stays stopped"},
	} {
		if ev := <-got; ev.Event != "notice" || *ev.Notice != want {
			t.Errorf("streamed %+v, want the notice %+v", ev, want)
		}
	}
	wait, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	for _, want := range added {
		d.tell(ctx, "linux-4", engine.Told{Kind: "agent_typing", Parts: []string{"review-bot types"}})
		d.tell(ctx, "linux-4", engine.Told{Kind: want.Kind, Parts: strings.Split(want.Text, " / ")})
		select {
		case ev := <-got:
			if ev.Event != "notice" || *ev.Notice != want {
				t.Errorf("streamed %+v, want the notice %+v with the HUD's icon", ev, want)
			}
		case <-wait.Done():
			t.Fatalf("the notice %s was never streamed", want.Kind)
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
