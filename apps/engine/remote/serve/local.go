package serve

import (
	"context"
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
)

type local struct {
	registry *call.Registry
	thisMac  machine.Profile
	version  string
}

func Local(registry *call.Registry, thisMac machine.Profile, version string) Calls {
	// value: a machine with no VM to relay to: every frame runs its calls here.
	return local{registry: registry, thisMac: thisMac, version: version}
}

func (l local) Hello() api.Hello {
	h := api.Hello{Version: l.version, Calls: []string{}, Kind: l.thisMac.Kind}
	for _, m := range l.registry.Metas() {
		h.Calls = append(h.Calls, m.Name)
	}
	return h
}

func (l local) Invoke(ctx context.Context, _ machine.Ref, name string, in json.RawMessage, p call.Progress) (any, error) {
	v, err := l.registry.Input(name, in)
	if err != nil {
		return nil, err
	}
	return l.registry.Run(ctx, l.thisMac, name, v, p)
}

func (l local) Watch(ctx context.Context, _ machine.Ref, name string, in json.RawMessage, p call.Progress) (<-chan call.Update[any], error) {
	v, err := l.registry.Input(name, in)
	if err != nil {
		return nil, err
	}
	return l.registry.Stream(ctx, l.thisMac, name, v, p)
}
