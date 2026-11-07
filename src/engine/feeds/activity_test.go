package feeds_test

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

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
			w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
			macs := map[string]string{"infra": mini, "linux-1": thisMac}
			live := &feeds.Live{Calls: w, Clock: newSteps(), ThisMac: thisMac, MacOf: func(_ context.Context, vm string) (string, bool) {
				mac, ok := macs[vm]
				return mac, ok
			}}
			o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac}, Live: live}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			p := until(t, o.Activity()(ctx, engine.FeedInput{Subject: c.subject, Changed: make(chan struct{}), Refresh: make(chan struct{})}), "the machine's activity", func(p page.Page) bool { return len(p.Sections) > 0 && p.Context.Machine == c.machine })
			w.mu.Lock()
			defer w.mu.Unlock()
			reads := w.reads[c.on]
			if len(reads) == 0 || reads[0].Of != c.of || reads[0].Graph != c.window {
				t.Errorf("%s's activity was read %+v on %s (all %v); want its %v read there, of %q", c.machine, reads, c.on, w.reads, c.window, c.of)
			}
			if p.ID != "menubar.activity" {
				t.Errorf("the page is %s; want menubar.activity", p.ID)
			}
		})
	}
}
