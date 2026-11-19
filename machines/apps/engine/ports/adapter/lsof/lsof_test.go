package lsof

import (
	"context"
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
