package wire

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

// value: a run that outlives this is a hook that ignored its own budget; it
// is a safety net far above the budget, never an assertion on elapsed time.
const safety = 10 * time.Second

type journal struct {
	hang  bool
	ended chan error
}

func (j journal) Invoke(ctx context.Context, _ coremachine.Ref, _ string, _, _ any, _ call.Progress) error {
	if j.hang {
		<-ctx.Done()
		j.ended <- ctx.Err()
		return ctx.Err()
	}
	panic("the journal could not be written")
}

func (journal) Watch(context.Context, coremachine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("a record is not followed")
}

// invariant: record runs the hook and returns what it printed, once it returned while
// the safety deadline was still ahead: the hook let go by its own budget.
func record(t *testing.T, calls call.Router, stdin io.Reader, args ...string) (said string, within context.Context) {
	t.Helper()
	var out bytes.Buffer
	env := kit.Env{Calls: calls, Stdin: stdin, Stdout: &out}
	cmd := agentsCommand(env, page.Context{}, agentsDeps{})
	cmd.SetArgs(append([]string{"record", "--app", "claudecode"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	ctx, cancel := context.WithTimeout(t.Context(), safety)
	t.Cleanup(cancel)
	finished := make(chan error, 1)
	go func() { finished <- cmd.ExecuteContext(ctx) }()
	select {
	case err := <-finished:
		if err != nil {
			t.Errorf("the hook's run failed with %v; a hook never fails the agent", err)
		}
	case <-ctx.Done():
		t.Fatalf("the hook still ran after %v; it ignored its own budget", safety)
	}
	if ctx.Err() != nil {
		t.Fatalf("the hook returned only once the %v safety deadline passed; it ignored its own budget", safety)
	}
	return out.String(), ctx
}

func TestAHookWhoseEventNeverEndsIsLetGoWithinItsBudget(t *testing.T) {
	stdin, open := io.Pipe()
	defer func() { _ = open.Close() }()
	said, _ := record(t, journal{}, stdin)
	if said != "" {
		t.Errorf("the hook printed %q; an agent reads a hook's output", said)
	}
}

func TestAJournalThatNeverAnswersOrFailsLeavesTheAgentAlone(t *testing.T) {
	for name, j := range map[string]journal{
		"never answers": {hang: true, ended: make(chan error, 1)},
		"fails":         {},
	} {
		said, ctx := record(t, j, nil, `{"kind":"began","session":"s-1"}`)
		if said != "" {
			t.Errorf("a journal that %s: the hook printed %q", name, said)
		}
		if !j.hang {
			continue
		}
		select {
		case err := <-j.ended:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("a journal that %s: its call ended with %v; the hook's budget should have fired", name, err)
			}
		case <-ctx.Done():
			t.Errorf("a journal that %s: its call never ended", name)
		}
	}
}
