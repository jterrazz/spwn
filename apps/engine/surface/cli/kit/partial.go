package kit

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

type PartialFailure struct {
	Failed, Total int
	Noun          string
	// value: what befell the failed ones, "failed" when empty.
	Verb      string
	FirstHint string
}

func (p PartialFailure) Error() string {
	words := []string{strconv.Itoa(p.Failed), "of", strconv.Itoa(p.Total)}
	if p.Noun != "" {
		words = append(words, p.Noun)
	}
	return strings.Join(append(words, cmp.Or(p.Verb, "failed")), " ")
}

func (p PartialFailure) As(target any) bool {
	t, ok := target.(**fail.Error)
	if ok {
		*t = failed(p.Error(), p.FirstHint, nil)
	}
	return ok
}
