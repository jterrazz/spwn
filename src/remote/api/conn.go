package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
)

const maxFrame = 64 << 20

type Conn struct {
	// invariant: the client half of one connection: requests go out under
	// their own id, a reader hands each answer to the request it names; when
	// the far side stops answering, every request still waiting gets Broken's
	// error.
	w      io.Writer
	close  func() error
	broken func(error) error

	mu      sync.Mutex
	next    int
	waiting map[string]*pending
	gone    error
	hello   *Hello
	ended   chan struct{}
	// value: this binary's version, which its hello names to the far side.
	version string
}

func NewConn(r io.Reader, w io.Writer, closer func() error, broken func(error) error, version string) *Conn {
	c := &Conn{w: w, close: closer, broken: broken, waiting: map[string]*pending{}, ended: make(chan struct{}), version: version}
	go c.read(r)
	return c
}

func (c *Conn) read(r io.Reader) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64<<10), maxFrame)
	for s.Scan() {
		var f Frame
		if json.Unmarshal(s.Bytes(), &f) != nil {
			continue
		}
		c.mu.Lock()
		p := c.waiting[f.ID]
		c.mu.Unlock()
		if p != nil {
			select {
			case p.ch <- f:
			case <-p.quit:
			}
		}
	}
	err := s.Err()
	if err == nil {
		err = io.EOF
	}
	if c.broken != nil {
		err = c.broken(err)
	}
	c.mu.Lock()
	c.gone = err
	waiting := c.waiting
	c.waiting = map[string]*pending{}
	c.mu.Unlock()
	for _, p := range waiting {
		close(p.ch)
	}
	close(c.ended)
}

func (c *Conn) send(f Frame) (id string, answers chan Frame, err error) {
	c.mu.Lock()
	if c.gone != nil {
		gone := c.gone
		c.mu.Unlock()
		return "", nil, gone
	}
	c.next++
	f.V, f.ID = Version, strconv.Itoa(c.next)
	ch := make(chan Frame, 16)
	c.waiting[f.ID] = &pending{ch: ch, quit: make(chan struct{})}
	var data []byte
	data, err = json.Marshal(f)
	if err == nil {
		_, err = c.w.Write(append(data, '\n'))
	}
	c.mu.Unlock()
	if err != nil {
		c.forget(f.ID)
		return "", nil, c.failure(err)
	}
	return f.ID, ch, nil
}

type pending struct {
	ch   chan Frame
	quit chan struct{}
}

func (c *Conn) forget(id string) {
	c.mu.Lock()
	if p, ok := c.waiting[id]; ok {
		close(p.quit)
		delete(c.waiting, id)
	}
	c.mu.Unlock()
}

func (c *Conn) failure(err error) error {
	select {
	case <-c.ended:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.gone
	default:
	}
	if c.broken != nil {
		return c.broken(err)
	}
	return err
}

func (c *Conn) lost() error {
	<-c.ended
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gone
}

func (c *Conn) Hello(ctx context.Context) (Hello, error) {
	// invariant: asked once per connection, so a far side upgraded since the
	// last connection is read again.
	c.mu.Lock()
	if c.hello != nil {
		h := *c.hello
		c.mu.Unlock()
		return h, nil
	}
	c.mu.Unlock()
	id, ch, err := c.send(Frame{Hello: &Hello{Version: c.version}})
	if err != nil {
		return Hello{}, err
	}
	defer c.forget(id)
	select {
	case f, ok := <-ch:
		if !ok {
			return Hello{}, c.lost()
		}
		if f.Error != nil {
			return Hello{}, f.Error.Err()
		}
		if f.Hello == nil {
			return Hello{}, errors.New("api: the far side answered no hello")
		}
		c.mu.Lock()
		c.hello = f.Hello
		c.mu.Unlock()
		return *f.Hello, nil
	case <-ctx.Done():
		return Hello{}, ctx.Err()
	}
}

func (c *Conn) Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (json.RawMessage, error) {
	id, ch, err := c.send(Frame{Call: name, Target: TargetOf(target), In: in})
	if err != nil {
		return nil, err
	}
	defer c.forget(id)
	for {
		select {
		case f, ok := <-ch:
			if !ok {
				return nil, c.lost()
			}
			switch {
			case f.Progress != nil:
				Report(p, f.Progress)
			case f.Error != nil:
				return nil, f.Error.Err()
			default:
				return f.Out, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *Conn) Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[json.RawMessage], error) {
	id, ch, err := c.send(Frame{Watch: name, Target: TargetOf(target), In: in})
	if err != nil {
		return nil, err
	}
	out := make(chan call.Update[json.RawMessage])
	go func() {
		defer close(out)
		defer c.forget(id)
		for {
			var u call.Update[json.RawMessage]
			select {
			case f, ok := <-ch:
				switch {
				case !ok:
					u.Err = c.lost()
				case f.Progress != nil:
					Report(p, f.Progress)
					continue
				case f.Done:
					return
				case f.Error != nil:
					u.Err = f.Error.Err()
				default:
					u.Value = f.Out
				}
			case <-ctx.Done():
				return
			}
			select {
			case out <- u:
			case <-ctx.Done():
				return
			}
			if u.Err != nil {
				return
			}
		}
	}()
	return out, nil
}

func Report(p call.Progress, f *Progress) {
	// value: a progress frame told to the caller's Progress.
	if p == nil {
		return
	}
	if f.Done {
		var err error
		if f.Error != nil {
			err = f.Error.Err()
		}
		p.Done(f.Task, err)
		return
	}
	p.Step(f.Task, f.Detail)
}

func (c *Conn) Close() error {
	if c.close == nil {
		return nil
	}
	return c.close()
}
