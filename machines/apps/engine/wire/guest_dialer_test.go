package wire

import (
	"context"
	"io"
	"io/fs"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type exitingGuest struct{ done chan struct{} }

func (exitingGuest) Exec(_ context.Context, c contract.GuestCommand) (contract.GuestResult, error) {
	_, _ = c.Stdout.Write([]byte("asked\n"))
	return contract.GuestResult{Code: 3}, nil
}
func (exitingGuest) Forward(context.Context, string, int) (io.ReadWriteCloser, error) {
	return nil, context.Canceled
}
func (exitingGuest) Put(context.Context, string, []byte, fs.FileMode) error { return nil }
func (exitingGuest) Close() error                                           { return nil }
func (g exitingGuest) Done() <-chan struct{}                                { return g.done }

func TestAnAttachedCommandAnswersItsExitCodeOnceItsOutputEnds(t *testing.T) {
	s, err := engineGuest{exitingGuest{done: make(chan struct{})}}.Attach(t.Context(), "sign-in")
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(s)
	if err != nil || string(out) != "asked\n" {
		t.Fatalf("output = %q (%v)", out, err)
	}
	if code, err := s.Wait(); code != 3 || err != nil {
		t.Errorf("Wait = %d, %v; want the command's own 3", code, err)
	}
}
