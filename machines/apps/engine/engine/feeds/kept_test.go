package feeds

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func TestAMachineThatDoesNotAnswerKeepsNoLastActivity(t *testing.T) {
	was := &activity.Report{Source: activity.FromEngine}
	read := activity.Report{Source: activity.FromDirect}
	silent := fail.New(fail.Unreachable, call.Unreachable, "mac-mini", nil)
	if got := keptAfter(was, read, nil); got == nil || got.Source != activity.FromDirect {
		t.Errorf("an answered read keeps %+v, want that read", got)
	}
	if got := keptAfter(was, read, silent); got != nil {
		t.Errorf("a machine that does not answer keeps %+v, want none", got)
	}
	if got := keptAfter(was, read, errors.New("busy")); got != was {
		t.Errorf("a read that failed otherwise keeps %+v, want the page's last", got)
	}
}
