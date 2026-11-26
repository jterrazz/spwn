package call

import "context"

type farKey struct{}

func WithFar(ctx context.Context) context.Context {
	// value: the call came from another machine: the transport that read
	// its frame from afar marks it, and every hop after carries the mark.
	return context.WithValue(ctx, farKey{}, true)
}

func Far(ctx context.Context) bool {
	// invariant: a mark is only ever set, never cleared: a frame that says
	// nothing of it leaves the mark its transport gave.
	far, _ := ctx.Value(farKey{}).(bool)
	return far
}
