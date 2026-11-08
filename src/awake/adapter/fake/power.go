package fake

import (
	"context"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/awake/contract"
	"github.com/jterrazz/studio-station/src/world"
)

const thisMac = "this-mac"

type Power struct {
	// invariant: the power of one Mac of the world: its reading is the world's,
	// so an event of the world moves it; a Mac with no power slice has no
	// power management.
	w        *world.World
	mac      string
	mu       sync.Mutex
	watchers []chan awake.Event
}

func PowerOf(w *world.World, mac string) *Power { return &Power{w: w, mac: mac} }

func NewPower(r awake.Reading) *Power {
	w := &world.World{ThisMac: thisMac}
	w.Mac(thisMac).Power = &world.Power{Reading: r}
	return PowerOf(w, thisMac)
}

func NewNoPower() *Power {
	w := &world.World{ThisMac: thisMac}
	w.Mac(thisMac)
	return PowerOf(w, thisMac)
}

func (p *Power) slice() *world.Power { return p.w.Mac(p.mac).Power }

func (p *Power) Capabilities() contract.PowerCaps {
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	if s == nil {
		return contract.PowerCaps{}
	}
	return contract.PowerCaps{Battery: s.HasBattery, Lid: s.Lid != "", Thermal: true}
}

func (p *Power) Read() (awake.Reading, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	if s == nil {
		return awake.Reading{}, contract.ErrUnsupported
	}
	return s.Reading, nil
}

func (p *Power) Sleeping() bool {
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	return s != nil && s.Sleeping
}

func (p *Power) Set(r awake.Reading) {
	// invariant: Set moves the reading and tells every watch which fields moved.
	p.w.Lock()
	s := p.slice()
	if s == nil {
		s = &world.Power{}
		p.w.Mac(p.mac).Power = s
	}
	was := s.Reading
	s.Reading = r
	p.w.Unlock()
	p.mu.Lock()
	watchers := slices.Clone(p.watchers)
	p.mu.Unlock()
	var kinds []string
	if was.Thermal != r.Thermal {
		kinds = append(kinds, "thermal")
	}
	if was.Source != r.Source {
		kinds = append(kinds, "source")
	}
	if was.Lid != r.Lid {
		kinds = append(kinds, "lid")
	}
	if was.HasBattery != r.HasBattery || was.Percent != r.Percent {
		kinds = append(kinds, "battery")
	}
	for _, w := range watchers {
		for _, k := range kinds {
			select {
			case w <- awake.Event{Kind: k}:
			default:
			}
		}
	}
}

func (p *Power) Watch(ctx context.Context) <-chan awake.Event {
	out := make(chan awake.Event, 8)
	p.w.Lock()
	none := p.slice() == nil
	p.w.Unlock()
	if none {
		close(out)
		return out
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.watchers = append(p.watchers, out)
	context.AfterFunc(ctx, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for i, w := range p.watchers {
			if w == out {
				p.watchers = append(p.watchers[:i], p.watchers[i+1:]...)
				close(out)
				return
			}
		}
	})
	return out
}

func (p *Power) Hold(name string) (func(), error) {
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	if s == nil {
		return nil, contract.ErrUnsupported
	}
	// invariant: one assertion per name, as a process holds it; taking it
	// again is the same one.
	if !slices.Contains(s.Holds, name) {
		s.Holds = append(s.Holds, name)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			p.w.Lock()
			defer p.w.Unlock()
			if s := p.slice(); s != nil {
				if i := slices.Index(s.Holds, name); i >= 0 {
					s.Holds = slices.Delete(s.Holds, i, i+1)
				}
			}
		})
	}, nil
}

func (p *Power) Clamshell(held bool) error {
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	if s == nil {
		return contract.ErrUnsupported
	}
	s.Clamshell = held
	return nil
}

func (p *Power) DisplaySleepNow(context.Context) error {
	// value: the request is recorded in the world, as the display sleeping.
	p.w.Lock()
	defer p.w.Unlock()
	s := p.slice()
	if s == nil {
		return contract.ErrUnsupported
	}
	s.DisplaySleeps++
	return nil
}
