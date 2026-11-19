package docker

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const budget = 5 * time.Second

type Space struct{ run local.Runner }

func New(run local.Runner) Space { return Space{run: run} }

func (s Space) Bytes(ctx context.Context) int64 {
	// invariant: a runtime that does not answer in time is unmeasured (-1);
	// one that is stopped or absent holds nothing station can count.
	out, err := s.run.Output(ctx, local.Cmd{Name: "docker", Args: []string{"system", "df", "--format", "{{.Size}}"}, Timeout: budget})
	if errors.Is(err, local.ErrTimeout) || ctx.Err() != nil {
		return -1
	}
	if err != nil || out.ExitCode != 0 {
		return 0
	}
	var sum int64
	for _, line := range strings.Split(out.Stdout, "\n") {
		sum += size(strings.TrimSpace(line))
	}
	return sum
}

// invariant: docker prints decimal units (kB, MB, GB, TB), the same ones station prints.
func size(s string) int64 {
	for _, unit := range []struct {
		suffix string
		scale  float64
	}{{"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"B", 1}} {
		if number, ok := strings.CutSuffix(s, unit.suffix); ok {
			value, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0
			}
			return int64(value * unit.scale)
		}
	}
	return 0
}
