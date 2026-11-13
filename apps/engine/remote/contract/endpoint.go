package contract

import (
	"context"
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Hello struct {
	Version string `json:"version"`
	// value: the names of the calls this binary serves.
	Calls []string `json:"calls"`
	// value: the kind of the machine that answers, as its own Profile says;
	// empty from a binary older than the field.
	Kind machine.Kind `json:"kind,omitempty"`
}

type EndpointDialer interface {
	// invariant: dials a Mac, never a VM: a VM's calls ride its Mac's endpoint.
	Dial(ctx context.Context, mac string) (Endpoint, error)
}

type Endpoint interface {
	Hello(ctx context.Context) (Hello, error)
	Invoke(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (json.RawMessage, error)
	Watch(ctx context.Context, target machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[json.RawMessage], error)
	Close() error
}
