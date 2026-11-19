package engine

import "time"

const (
	RecordSchema  = "station.engine-record/v1"
	RecordVersion = 1
	// value: the record's file, in the engine's state folder.
	RecordFile = "engine.json"

	InstalledSchema = "station.install-record/v1"
	// value: the public installer's record, in the engine's state folder
	// beside RecordFile and of the same shape: the installer writes it,
	// atomically, at the end of a successful install, naming the binary it
	// put in place; the engine only projects its contract. The record is
	// written only when a station binary was installed: a platform with no
	// station build gets spwn alone and exits 0, and an older record is left
	// byte for byte, never deleted. An install path with a control character
	// (below 0x20, or 0x7f) is refused before any write.
	InstalledFile = "installed.json"
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
