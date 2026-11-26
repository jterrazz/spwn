package call

import (
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

const (
	// value: an input over the most its call takes: a size that would make
	// the call allocate or loop without end.
	TooLarge fail.Code = "call.too-large"
)

func OverBound(name, in, most string) error {
	// value: the refusal of an input past its maximum, naming the input and
	// the most it takes, in the words the caller reads.
	return fail.New(fail.Refused, TooLarge, name, map[string]string{"in": in, "max": most})
}
