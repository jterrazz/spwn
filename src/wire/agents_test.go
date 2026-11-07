package wire

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type journal struct{ hang bool }

func (j journal) Invoke(ctx context.Context, _ coremachine.Ref, _ string, _, _ any, _ call.Progress) error {
	if j.hang {
		<-ctx.Done()
		return ctx.Err()
	}
	panic("the journal could not be written")
}

func (journal) Watch(context.Context, coremachine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("a record is not followed")
}

func record(t *testing.T, calls call.Router, stdin io.Reader, args ...string) (took time.Duration, said string) {
	t.Helper()
	var out bytes.Buffer
	env := kit.Env{Calls: calls, Stdin: stdin, Stdout: &out}
	cmd := agentsCommand(env, page.Context{})
	cmd.SetArgs(append([]string{"record", "--app", "claudecode"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	clock := local.New(nil, nil, nil, nil).Clock
	start := clock.Now()
	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Errorf("the hook's run failed with %v; a hook never fails the agent", err)
	}
	return clock.Now().Sub(start), out.String()
}

func TestAHookWhoseEventNeverEndsIsLetGoWithinItsBudget(t *testing.T) {
	stdin, open := io.Pipe()
	defer func() { _ = open.Close() }()
	took, said := record(t, journal{}, stdin)
	if took > catalog.AgentHookBudget {
		t.Errorf("the hook waited %v on a stdin that never closes; its budget is %v", took, catalog.AgentHookBudget)
	}
	if said != "" {
		t.Errorf("the hook printed %q; an agent reads a hook's output", said)
	}
}

func TestAJournalThatNeverAnswersOrFailsLeavesTheAgentAlone(t *testing.T) {
	for name, j := range map[string]journal{"never answers": {hang: true}, "fails": {}} {
		took, said := record(t, j, nil, `{"kind":"began","session":"s-1"}`)
		if took > catalog.AgentHookBudget || said != "" {
			t.Errorf("a journal that %s: the hook took %v and printed %q", name, took, said)
		}
	}
}
