package macos

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const budget = 5 * time.Second

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Ephemeral struct{ run Commands }

func New(run Commands) *Ephemeral { return &Ephemeral{run: run} }

func (e *Ephemeral) Range(ctx context.Context) (from, to int, ok bool) {
	// outside: macOS answers each end of its outgoing connections' port range
	// to `sysctl -n net.inet.ip.portrange.first` and `.last`, one number each.
	from, fromOK := e.read(ctx, "net.inet.ip.portrange.first")
	to, toOK := e.read(ctx, "net.inet.ip.portrange.last")
	return from, to, fromOK && toOK && ports.PlausibleEphemeral(from, to)
}

func (e *Ephemeral) read(ctx context.Context, name string) (int, bool) {
	out, err := e.run.Output(ctx, local.Cmd{Name: "sysctl", Args: []string{"-n", name}, Timeout: budget})
	if err != nil || out.ExitCode != 0 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(out.Stdout))
	return n, err == nil
}
