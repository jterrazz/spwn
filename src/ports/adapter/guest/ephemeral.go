package guest

import (
	"context"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/ports"
)

type Ephemeral interface {
	// value: the range of ports a system hands its outgoing connections.
	Range(ctx context.Context) (from, to int, ok bool)
}

// value: the range the guest's system gives, else the default of its kind.
func (s *Agent) ephemeral(ctx context.Context) (from, to int) {
	if s.o.Ephemeral != nil {
		if from, to, ok := s.o.Ephemeral.Range(ctx); ok {
			return from, to
		}
	}
	if s.o.Kind == catalog.MacOS {
		return ports.MacOSEphemeralFrom, ports.MacOSEphemeralTo
	}
	return ports.LinuxEphemeralFrom, ports.LinuxEphemeralTo
}
