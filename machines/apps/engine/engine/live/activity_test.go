package live_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine/live"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// value: the activity reads asked, by the machine they ran on.
type charts struct {
	mu    sync.Mutex
	reads map[string][]activitycalls.ReadIn
}

func (c *charts) Invoke(context.Context, machine.Ref, string, any, any, call.Progress) error {
	return nil
}

func (c *charts) Watch(_ context.Context, target machine.Ref, _ string, in any, _ call.Progress) (<-chan call.Update[any], error) {
	at := target.Name
	if at == "" {
		at = thisMac
	}
	c.mu.Lock()
	read, _ := in.(activitycalls.ReadIn)
	c.reads[at] = append(c.reads[at], read)
	c.mu.Unlock()
	out := make(chan call.Update[any], 1)
	out <- call.Update[any]{Value: activity.Report{}}
	close(out)
	return out, nil
}

func TestTheActivityOfAnyMachineIsReadThroughTheEngineThatKeepsIt(t *testing.T) {
	for _, c := range []struct {
		subject, on, of string
		window          time.Duration
		machine         string
	}{
		{subject: "6m", on: thisMac, window: 6 * time.Minute, machine: thisMac},
		{subject: mini + ":24h", on: mini, window: 24 * time.Hour, machine: mini},
		{subject: "infra:6m", on: mini, of: "infra", window: 6 * time.Minute, machine: "infra"},
		{subject: "linux-1", on: thisMac, of: "linux-1", window: time.Hour, machine: "linux-1"},
	} {
		t.Run(c.subject, func(t *testing.T) {
			w := &charts{reads: map[string][]activitycalls.ReadIn{}}
			macs := map[string]string{"infra": mini, "linux-1": thisMac}
			rings := &live.Live{Calls: w, ThisMac: thisMac, MacOf: func(_ context.Context, vm string) (string, bool) {
				mac, ok := macs[vm]
				return mac, ok
			}}
			a := live.Activity{Calls: w, Clock: &moving{}, ThisMachine: page.Context{Machine: thisMac}, Live: rings}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			got, ok := <-a.Page(ctx, c.subject)
			if !ok {
				t.Fatal("no page of the machine's activity")
			}
			doc, _ := got.(page.Document)
			p := doc.Page
			w.mu.Lock()
			defer w.mu.Unlock()
			reads := w.reads[c.on]
			if len(reads) == 0 || reads[0].Of != c.of || reads[0].Graph != c.window {
				t.Errorf("%s's activity was read %+v on %s (all %v); want its %v read there, of %q", c.machine, reads, c.on, w.reads, c.window, c.of)
			}
			if p.ID != "menubar.activity" || p.Context.Machine != c.machine {
				t.Errorf("the page is %s of %s; want menubar.activity of %s", p.ID, p.Context.Machine, c.machine)
			}
		})
	}
}
