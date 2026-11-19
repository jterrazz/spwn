package engine

import "time"

const (
	RecordSchema  = "station.engine-record/v1"
	RecordVersion = 1
)

type Record struct {
	// invariant: a later shape of the record a window reads to find the
	// station binary while the engine does not answer bumps V, never changes
	// this one.
	V          int       `json:"v"`
	Executable string    `json:"executable"`
	Version    string    `json:"version"`
	At         time.Time `json:"at"`
}

type Records interface {
	// value: the binary the engine started from, kept beside its socket and
	// replaced whole at each start; the engine is its one writer.
	Write(r Record) error
}
