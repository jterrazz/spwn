package docker

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Containers struct{ run local.Runner }

func New(run local.Runner) *Containers { return &Containers{run: run} }

var errAnswered = errors.New("docker ps did not answer")

func (p *Containers) Read(ctx context.Context) ([]ports.Announced, error) {
	// value: nil and no error when docker is absent: no container publishes.
	if _, ok := p.run.LookPath("docker"); !ok {
		return nil, nil
	}
	out, err := p.run.Output(ctx, local.Cmd{Name: "docker", Args: []string{"ps", "--format", "{{json .}}"}, Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errAnswered
	}
	return ParsePS([]byte(out.Stdout)), nil
}

type row struct {
	Names string `json:"Names"`
	Ports string `json:"Ports"`
}

func ParsePS(data []byte) []ports.Announced {
	byPort := map[int]ports.Announced{}
	for _, line := range strings.Split(string(data), "\n") {
		var r row
		if strings.TrimSpace(line) == "" || json.Unmarshal([]byte(line), &r) != nil {
			continue
		}
		for _, entry := range strings.Split(r.Ports, ",") {
			mapped, _, ok := strings.Cut(strings.TrimSpace(entry), "->")
			if !ok {
				continue
			}
			i := strings.LastIndex(mapped, ":")
			if i < 0 {
				continue
			}
			port, err := strconv.Atoi(mapped[i+1:])
			if err != nil {
				continue
			}
			ip := strings.Trim(mapped[:i], "[]")
			if _, seen := byPort[port]; !seen {
				byPort[port] = ports.Announced{Port: port, IP: ip, Process: r.Names, Source: ports.ByContainer}
			}
		}
	}
	for port, l := range byPort {
		if l.IP == "::" {
			l.IP = "0.0.0.0"
			byPort[port] = l
		}
	}
	return ports.SortedAnnounced(byPort)
}
