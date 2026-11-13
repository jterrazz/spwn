package lsof

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

type Lsof struct{ run Commands }

func New(run Commands) *Lsof { return &Lsof{run: run} }

func (l *Lsof) Read(ctx context.Context) ([]ports.Listener, error) {
	out, err := l.run.Output(ctx, local.Cmd{Name: "lsof", Args: []string{"-iTCP", "-sTCP:LISTEN", "-P", "-n", "-Fpcn"}, Timeout: budget})
	if err != nil {
		return nil, err
	}
	return parse(out.Stdout), nil
}

// outside: lsof -F prints one field per line, p<pid>, c<command>, n<address>,
// a wildcard address as *.
func parse(text string) []ports.Listener {
	var out []ports.Listener
	var pid int
	var command string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		field := line[1:]
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(field)
		case 'c':
			command = field
		case 'n':
			i := strings.LastIndex(field, ":")
			if i < 0 {
				continue
			}
			port, err := strconv.Atoi(field[i+1:])
			if err != nil {
				continue
			}
			bind := strings.Trim(field[:i], "[]")
			if bind == "*" {
				bind = "0.0.0.0"
			}
			out = append(out, ports.Listener{Port: port, Bind: bind, Process: command, PID: pid})
		}
	}
	return out
}
