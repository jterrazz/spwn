package engine

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	EventExposed = "port_exposed"
)

type ExposureKept struct {
	// value: what the exposure job knows between two commands: when it last
	// read the listeners, what each VM listened on then, and the red ports
	// it already told.
	At      time.Time                   `json:"at,omitzero"`
	Scanned map[string][]ports.Listener `json:"scanned,omitempty"`
	Red     []string                    `json:"red,omitempty"`
}
