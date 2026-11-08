package process

import (
	"context"
	"sort"
	"time"

	"github.com/jterrazz/studio-station/src/engine"
)

const backEvery = 30 * time.Second

// invariant: SPEC B5.14: a Mac the owner asked to be told of is asked at
// most every 30 s, and only while the ask stands; the moment it answers,
// mac_back is told once and the ask is cleared.
func (d *Engine) tellBack(ctx context.Context) {
	if d.o.TellBack == nil || d.o.Answers == nil {
		return
	}
	now := d.h.Now()
	if !d.askedBack.IsZero() && now.Sub(d.askedBack) < backEvery {
		return
	}
	d.askedBack = now
	armed := d.o.TellBack()
	if len(armed) == 0 {
		return
	}
	macs := make([]string, 0, len(armed))
	for mac := range armed {
		macs = append(macs, mac)
	}
	sort.Strings(macs)
	for _, mac := range macs {
		if !d.o.Answers(ctx, mac) {
			continue
		}
		if d.o.ToldBack != nil && d.o.ToldBack(mac) != nil {
			continue
		}
		label := mac
		if d.o.Label != nil {
			label = d.o.Label(mac)
		}
		d.tell(ctx, mac, engine.Told{Kind: engine.NoticeMacBack, Parts: []string{label + " is back"}})
	}
}
