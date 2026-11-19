package contract

import (
	"context"
	"encoding/json"
	"slices"

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
	// value: what its calls read of an input beyond their names
	// (call.Meta.Abilities); none from a binary older than the field.
	Abilities []string `json:"abilities,omitempty"`
}

func (h Hello) Lacks(name string, asks []string) string {
	// value: the first of the call and the abilities its input asks that
	// this binary does not serve; empty when it serves them all.
	if !slices.Contains(h.Calls, name) {
		return name
	}
	for _, a := range asks {
		if !slices.Contains(h.Abilities, a) {
			return a
		}
	}
	return ""
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
