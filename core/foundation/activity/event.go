// Package activity provides a system-wide event log for spwn.
//
// Events are appended to ~/.spwn/activity.jsonl and surface on the
// Observatory dashboard. Each event carries a natural-language phrase
// authored at emission time by the code that knows the semantics.
package activity

import "time"

// Type is the event type. Uses dotted namespace (subject.verb).
type Type string

const (
	TypeWorldSpawned     Type = "world.spawned"
	TypeWorldDestroyed   Type = "world.destroyed"
	TypeWorldSnapshot    Type = "world.snapshot"
	TypeWorldStateChange Type = "world.state_changed"
	TypeAgentCreated     Type = "agent.created"
	TypeAgentDeleted     Type = "agent.deleted"
	TypeAgentJoined      Type = "agent.joined"
	TypeAgentLeft        Type = "agent.left"
	TypeAgentDreamed     Type = "agent.dreamed"
	TypeAgentSlept       Type = "agent.slept"
	TypeAgentForked      Type = "agent.forked"
	TypeAgentTalked      Type = "agent.talked"
	TypeArchitectStarted Type = "architect.started"
	TypeArchitectStopped Type = "architect.stopped"
	TypeArchitectTalked  Type = "architect.talked"
	TypeSessionEnded     Type = "world.session_ended"
)

// Event is a single activity entry.
type Event struct {
	ID         string         `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	Type       Type           `json:"type"`
	Actor      string         `json:"actor"`
	Verb       string         `json:"verb"`
	Target     string         `json:"target,omitempty"`
	Phrase     string         `json:"phrase"`
	WorldID    string         `json:"world_id,omitempty"`
	AgentID    string         `json:"agent_id,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
	CostUSD    float64        `json:"cost_usd,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
