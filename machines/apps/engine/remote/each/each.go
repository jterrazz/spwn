package each

import (
	"context"
	"sync"
	"time"
)

const (
	// value: at most this many Macs are asked at once.
	Width = 4
	// value: what one Mac is given to answer before it is a finding.
	Budget = 5 * time.Second
)

type Result[T any] struct {
	Mac   string
	Value T
	Err   error
}

func Do[T any](ctx context.Context, macs []string, ask func(ctx context.Context, mac string) (T, error)) <-chan Result[T] {
	// invariant: the All machines fan-out: Width Macs at a time, each within
	// its Budget, every answer handed on as it comes; a Mac that does not
	// answer in time is its own result's error, never the others' delay; a
	// reader that stops early never holds a Mac's goroutine.
	out := make(chan Result[T], len(macs))
	lanes := make(chan struct{}, Width)
	var wg sync.WaitGroup
	for _, mac := range macs {
		wg.Go(func() {
			select {
			case lanes <- struct{}{}:
			case <-ctx.Done():
				out <- Result[T]{Mac: mac, Err: ctx.Err()}
				return
			}
			defer func() { <-lanes }()
			actx, cancel := context.WithTimeout(ctx, Budget)
			defer cancel()
			v, err := answer(actx, mac, ask)
			out <- Result[T]{Mac: mac, Value: v, Err: err}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func answer[T any](ctx context.Context, mac string, ask func(context.Context, string) (T, error)) (T, error) {
	type got struct {
		v   T
		err error
	}
	done := make(chan got, 1)
	go func() {
		v, err := ask(ctx, mac)
		done <- got{v, err}
	}()
	select {
	case g := <-done:
		return g.v, g.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

func All[T any](ctx context.Context, macs []string, ask func(ctx context.Context, mac string) (T, error)) map[string]Result[T] {
	// value: Do's results once every Mac has answered or run out of time.
	out := make(map[string]Result[T], len(macs))
	for r := range Do(ctx, macs, ask) {
		out[r.Mac] = r
	}
	return out
}
