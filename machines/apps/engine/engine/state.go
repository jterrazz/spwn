package engine

import "encoding/json"

const (
	// value: the stream engine.state speaks, and the revision of it this
	// engine speaks, judged by FitOf against the client's.
	StateStream   = "station.state/v1"
	StateRevision = 2
)

const (
	StateOpened   = "opened"
	StateSnapshot = "snapshot"
	StateChanged  = "changed"
	StateTick     = "tick"
)

type StateIn struct {
	// value: the slices asked: an exact key (`mac:macbook-pro`), every key
	// of a family (`overview:*`), or `local`, every key this Mac exports.
	Keys []string `json:"keys"`
	// value: the revision the client holds, of the engine whose epoch it
	// names: inside the store's ring, the first frame is the keys changed
	// since, else a snapshot.
	Since *StateSince `json:"since,omitempty"`
	// value: true streams every change; false answers the opening and one
	// snapshot or change, then ends.
	Hold bool `json:"hold,omitempty"`
	// value: the revision of station.state/v1 the client reads.
	Revision int `json:"revision"`
	// value: the pace of whoever reads the stream: a held stream holds the
	// engine's readers at their attached pace unless it says idle, as
	// another Mac's engine does while no client is attached to it.
	Pace string `json:"pace,omitempty"`
}

const (
	PaceAttached = "attached"
	PaceIdle     = "idle"
)

type Held map[string]map[string]SliceOut

type StateSince struct {
	Epoch string `json:"epoch"`
	Rev   uint64 `json:"rev"`
}

type StateFrame struct {
	// value: one of opened, snapshot, changed, tick.
	Event string `json:"event"`
	// value: the opening's: the stream, the revisions the engine speaks, its
	// epoch and the verdict on the client's revision.
	Stream string `json:"stream,omitempty"`
	State  int    `json:"state,omitempty"`
	Page   int    `json:"page,omitempty"`
	Epoch  string `json:"epoch,omitempty"`
	Fit    *Fit   `json:"fit,omitempty"`
	// value: the store's revision and its clock as of this frame.
	Rev uint64 `json:"rev"`
	At  string `json:"at,omitempty"`
	// value: a snapshot's every asked slice, a change's slices that moved,
	// and the keys gone since the last frame.
	Slices map[string]SliceOut `json:"slices,omitempty"`
	Gone   []string            `json:"gone,omitempty"`
}

type SliceOut struct {
	// value: the slice's value as an answer encodes it; absent from a change
	// whose hash the client already holds, only its state moved.
	Value  json.RawMessage `json:"value,omitempty"`
	Hash   string          `json:"hash,omitempty"`
	ReadAt string          `json:"read_at,omitempty"`
	State  string          `json:"state"`
	Why    string          `json:"why,omitempty"`
}
