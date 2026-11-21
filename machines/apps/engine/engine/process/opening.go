package process

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// invariant: each page a watch streams logs one line of its opening, once
// its first full page went out or once the watch ended first: how long
// after the watch opened its first page went out, its first page with a
// header, its first full page, how many pages went out before the full
// one, and how many commands the engine ran meanwhile. A full page has a
// header and no value or mark still pending, and no loading state.
type opening struct {
	mu       sync.Mutex
	started  time.Time
	commands int64
	now      func() time.Time
	ran      func() int64
	log      func(subject, message string)
	pages    map[string]*opened
}

type opened struct {
	first, header, full time.Duration
	frames              int
	said                bool
}

func (d *Engine) opening(pages []string) *opening {
	o := &opening{now: d.h.Now, ran: d.o.Commands, log: d.logf, pages: map[string]*opened{}}
	o.started = o.now()
	o.commands = o.count()
	for _, id := range pages {
		o.pages[id] = &opened{first: -1, header: -1, full: -1}
	}
	return o
}

func (o *opening) count() int64 {
	if o.ran == nil {
		return 0
	}
	return o.ran()
}

func (o *opening) wrote(id string, p json.RawMessage) {
	o.mu.Lock()
	defer o.mu.Unlock()
	page, ok := o.pages[id]
	if !ok || page.said {
		return
	}
	at := o.now().Sub(o.started)
	page.frames++
	if page.first < 0 {
		page.first = at
	}
	head, full := pageShape(p)
	if head && page.header < 0 {
		page.header = at
	}
	if full {
		page.full = at
		o.say(id, page)
	}
}

func (o *opening) end() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for id, page := range o.pages {
		if !page.said {
			o.say(id, page)
		}
	}
}

func (o *opening) say(id string, page *opened) {
	page.said = true
	o.log("watch", fmt.Sprintf("%s first=%s header=%s full=%s frames=%d commands=%d", id, ms(page.first), ms(page.header), ms(page.full), page.frames, o.count()-o.commands))
}

func ms(d time.Duration) string {
	if d < 0 {
		return "-"
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func pageShape(p json.RawMessage) (header, full bool) {
	var shape struct {
		Header *json.RawMessage `json:"header"`
		State  string           `json:"state"`
	}
	if json.Unmarshal(p, &shape) != nil {
		return false, false
	}
	header = shape.Header != nil
	pending := bytes.Contains(p, []byte(`"mark":"pending"`)) || bytes.Contains(p, []byte(`"kind":"pending"`))
	return header, header && !pending && !strings.EqualFold(shape.State, "loading")
}
