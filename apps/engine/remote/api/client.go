package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const ConnectBudget = 2 * time.Second

type Dial func(ctx context.Context) (io.ReadWriteCloser, error)

type Client struct{ dial Dial }

func NewClient(dial Dial) Client { return Client{dial: dial} }

// invariant: a server that is absent, refuses, or does not accept within the
// connect budget is down: Down, never another error.
func (c Client) open(ctx context.Context, f Frame) (io.ReadWriteCloser, *bufio.Reader, error) {
	cctx, cancel := context.WithTimeout(ctx, ConnectBudget)
	conn, err := c.dial(cctx)
	cancel()
	if err != nil {
		return nil, nil, &Down{}
	}
	f.V, f.ID = Version, "1"
	data, err := json.Marshal(f)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		_ = conn.Close()
		return nil, nil, &Down{}
	}
	return conn, bufio.NewReaderSize(conn, 64*1024), nil
}

func encodeIn(in any) (json.RawMessage, error) {
	if in == nil {
		return nil, nil
	}
	return json.Marshal(in)
}

func (c Client) Call(ctx context.Context, name string, target machine.Ref, in, out any, p call.Progress) error {
	raw, err := encodeIn(in)
	if err != nil {
		return err
	}
	conn, r, err := c.open(ctx, Frame{Call: name, Target: TargetOf(target), In: raw})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	frames := make(chan Frame)
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		read(r, frames, ctx.Done())
	}()
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close()
			<-ended
			return ctx.Err()
		case f, ok := <-frames:
			if !ok {
				// invariant: a connection that ends before its answer is a
				// server gone: down.
				return &Down{}
			}
			switch {
			case f.Progress != nil:
				Report(p, f.Progress)
			case f.Error != nil:
				return f.Error.Err()
			default:
				if out != nil && len(f.Out) > 0 {
					return json.Unmarshal(f.Out, out)
				}
				return nil
			}
		}
	}
}

func read(r *bufio.Reader, into chan<- Frame, stop <-chan struct{}) {
	defer close(into)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var f Frame
		if json.Unmarshal(line, &f) != nil {
			continue
		}
		select {
		case into <- f:
		case <-stop:
			return
		}
	}
}

func (c Client) Watch(ctx context.Context, name string, in any) (<-chan json.RawMessage, error) {
	// invariant: a watch is its own connection, held until ctx ends or the
	// server goes; a refusal comes back at once as the error, else each out is
	// sent as it came, and the channel closes when the stream does.
	raw, err := encodeIn(in)
	if err != nil {
		return nil, err
	}
	conn, r, err := c.open(ctx, Frame{Watch: name, In: raw})
	if err != nil {
		return nil, err
	}
	frames := make(chan Frame)
	go read(r, frames, ctx.Done())
	first, ok := <-frames
	if !ok {
		_ = conn.Close()
		return nil, &Down{}
	}
	if first.Error != nil {
		_ = conn.Close()
		return nil, first.Error.Err()
	}
	out := make(chan json.RawMessage)
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	go func() {
		defer close(out)
		for f := first; ; {
			if f.Error != nil {
				return
			}
			if len(f.Out) > 0 {
				select {
				case out <- f.Out:
				case <-ctx.Done():
					return
				}
			}
			if f, ok = <-frames; !ok {
				return
			}
		}
	}()
	return out, nil
}
