package engine

import (
	"context"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type AgentHeldError struct {
	Holder model.Agent
	Freed  func(ctx context.Context) error
}

func (e *AgentHeldError) Error() string {
	return "the ssh agent socket is held by " + e.Holder.Holder() + ", not this engine; it is served once freed: " + e.Holder.Free
}
