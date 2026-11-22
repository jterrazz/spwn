package lsof

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

type recorded struct{ stdout string }

func (r recorded) Output(context.Context, local.Cmd) (local.Output, error) {
	return local.Output{Stdout: r.stdout}, nil
}

func TestLsofKeepsTheListenersContract(t *testing.T) {
	contracttest.Listeners(t, func(*testing.T) contract.Listeners {
		return New(recorded{stdout: "p100\ncnode\nn*:3000\nn[::]:3000\np101\ncpostgres\nn127.0.0.1:5432\n"})
	})
}

type answered struct {
	asked *local.Cmd
	out   local.Output
}

func (a answered) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	*a.asked = c
	return a.out, nil
}

func TestTheTextsOfAProcessAreTheFilesItMapsUnderThePathsTheyHoldNow(t *testing.T) {
	var asked local.Cmd
	texts, err := New(answered{asked: &asked, out: local.Output{Stdout: "p9006\nftxt\nn/private/tmp/old-Spwn.app/Contents/MacOS/Spwn Engine\nftxt\nn/usr/lib/dyld\n"}}).Texts(context.Background(), 9006)
	if err != nil || !slices.Equal(texts, []string{"/private/tmp/old-Spwn.app/Contents/MacOS/Spwn Engine", "/usr/lib/dyld"}) {
		t.Fatalf("texts = %q, %v", texts, err)
	}
	if got := asked.Name + " " + strings.Join(asked.Args, " "); got != "lsof -a -p 9006 -d txt -Fn" {
		t.Errorf("asked %q", got)
	}
	if _, err := New(answered{asked: &asked, out: local.Output{ExitCode: 1}}).Texts(context.Background(), 9006); err == nil {
		t.Error("a process lsof does not find gave no error")
	}
}
