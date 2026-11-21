package store

import (
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"strconv"
	"time"
)

type State string

const (
	Fresh  State = "fresh"
	Stale  State = "stale"
	Unread State = "unread"
)

const (
	WhyReading     = "reading"
	WhyLate        = "late"
	WhyUnreachable = "unreachable"
	WhyFarTooOld   = "far-too-old"
)

type Slice struct {
	Key Key
	// value: the typed value, for the projections; nil while unread.
	Value any
	// value: call.Encode of Value, the bytes the wire carries.
	Raw  json.RawMessage
	Hash string
	// value: when Value was last read, as of Rev; zero while unread.
	ReadAt time.Time
	State  State
	// value: "" while fresh, else why it is not: one of the Why words or an
	// error's code.
	Why string
	// value: the revision at which Raw, State or Why last changed.
	Rev uint64
}

func hashOf(raw []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(raw)
	return hex.EncodeToString(h.Sum(nil))
}

type Epoch string

func EpochOf(started time.Time, pid int) Epoch {
	// invariant: one engine's life, the started_at and pid of its hello, so a
	// resume never crosses a restart.
	return Epoch(started.UTC().Format(time.RFC3339) + "/" + strconv.Itoa(pid))
}

type Cadence struct {
	Every  time.Duration
	Budget time.Duration
}

func (c Cadence) staleAfter() time.Duration {
	return 2*c.Every + c.Budget
}

type Snapshot struct {
	Epoch  Epoch
	Rev    uint64
	At     time.Time
	Slices map[Key]Slice
}
