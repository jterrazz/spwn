package api

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// invariant: a far side that takes the connection and never answers its
// hello is the connection's end, told by broken, within the hello budget,
// even when the caller's context carries no deadline.
func TestAFarSideSilentAfterConnectIsGoneWithinTheHelloBudget(t *testing.T) {
	out, far := io.Pipe()
	silent := errors.New("mac-mini does not answer")
	c := NewConn(out, io.Discard, far.Close, func(error) error { return silent }, "v0.1.0")
	c.helloBudget = 50 * time.Millisecond
	got := make(chan error, 1)
	go func() {
		_, err := c.Hello(context.Background())
		got <- err
	}()
	bound, stop := context.WithTimeout(t.Context(), 2*time.Second)
	defer stop()
	select {
	case err := <-got:
		if !errors.Is(err, silent) {
			t.Fatalf("Hello = %v, want the connection's end, %v", err, silent)
		}
	case <-bound.Done():
		_ = far.Close()
		t.Fatal("Hello still waits on a far side that never answers")
	}
}
