package process

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func TestAHeldAgentSocketReadsItsHolderAndIsServedOnceItsHolderLetsItGo(t *testing.T) {
	h := newHarness(t)
	freed := make(chan struct{})
	served := make(chan struct{})
	opens := 0
	h.d.o.Agent = func(context.Context) (func(context.Context) error, error) {
		opens++
		if opens == 1 {
			return nil, &engine.AgentHeldError{
				Holder: model.Agent{PID: 9006, Executable: "/private/tmp/old/Spwn.app/Contents/MacOS/Spwn Engine", Free: "kill 9006"},
				Freed: func(ctx context.Context) error {
					select {
					case <-freed:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
			}
		}
		return func(ctx context.Context) error {
			close(served)
			<-ctx.Done()
			return nil
		}, nil
	}
	var lines []string
	if err := h.d.Begin(t.Context(), func(line string) { lines = append(lines, line) }); err != nil {
		t.Fatal(err)
	}
	held := h.d.Hello().Agent
	if held.State != model.AgentHeld || held.PID != 9006 || !held.Unserved() || held.Fix() != "kill 9006" {
		t.Fatalf("the engine read the held socket as %+v", held)
	}
	if len(lines) == 0 || !strings.Contains(lines[0], "agent: the ssh agent socket is held by Spwn Engine (pid 9006), not this engine") {
		t.Fatalf("the engine said %q", lines)
	}
	close(freed)
	wait, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	select {
	case <-served:
	case <-wait.Done():
		t.Fatal("the engine did not serve the socket its holder let go")
	}
	if got := h.d.Hello().Agent; got.State != model.AgentServed || got.Unserved() {
		t.Fatalf("the engine read the freed socket as %+v", got)
	}
}

func TestAnAgentThatFailsToOpenReadsFailedAndAMacWithNoDeviceKeyReadsNone(t *testing.T) {
	for _, c := range []struct {
		story string
		err   error
		want  model.AgentState
	}{
		{"the socket cannot be made", errors.New("listen unix: address in use"), model.AgentFailed},
		{"the Mac cannot keep a device key", &contract.NoDeviceKeyError{Why: "no Secure Enclave"}, ""},
	} {
		h := newHarness(t)
		h.d.o.Agent = func(context.Context) (func(context.Context) error, error) { return nil, c.err }
		if err := h.d.Begin(t.Context(), func(string) {}); err != nil {
			t.Fatal(err)
		}
		if got := h.d.Hello().Agent; got.State != c.want {
			t.Errorf("%s: the agent reads %q, want %q", c.story, got.State, c.want)
		}
	}
}
