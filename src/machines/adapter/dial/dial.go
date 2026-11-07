package dial

import (
	"context"
	"strconv"

	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type Checker struct{}

var _ contract.CheckRunner = Checker{}

func (Checker) Run(ctx context.Context, t machines.CheckTarget) (machines.CheckAnswer, error) {
	status, took, err := local.Timed(ctx, local.Check{Dial: t.Dial, URL: t.URL, Name: t.Name, Verify: t.Verify, Budget: machines.CheckBudget})
	if err != nil {
		return machines.CheckAnswer{}, err
	}
	if status == 0 {
		return machines.CheckAnswer{Took: took}, nil
	}
	return machines.CheckAnswer{Status: strconv.Itoa(status), Took: took}, nil
}
