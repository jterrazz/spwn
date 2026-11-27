package calls

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAGraphWiderThanThirtyDaysIsRefusedBeforeAnyBucket(t *testing.T) {
	now := time.Date(2026, 11, 6, 4, 0, 0, 0, time.UTC)
	d := Deps{Meter: processes{}, Name: "mac-mini", Kind: machine.Computer, Now: func() time.Time { return now }}
	for graph, refused := range map[time.Duration]bool{
		0:                            false,
		-time.Hour:                   false,
		6 * time.Minute:              false,
		24 * time.Hour:               false,
		MaxGraph:                     false,
		MaxGraph + 1:                 true,
		365 * 24 * time.Hour:         true,
		time.Duration(math.MaxInt64): true,
	} {
		updates, err := Read.Run(t.Context(), d, ReadIn{Graph: graph, Quick: true}, nil)
		var e *fail.Error
		switch {
		case refused && (!errors.As(err, &e) || e.Code != call.TooLarge || e.Type != fail.Refused || e.Detail["in"] != "graph"):
			t.Errorf("a graph of %v answers %v, want it refused as %s", graph, err, call.TooLarge)
		case !refused && err != nil:
			t.Errorf("a graph of %v is refused: %v", graph, err)
		case !refused:
			for range updates {
			}
		}
	}
}
