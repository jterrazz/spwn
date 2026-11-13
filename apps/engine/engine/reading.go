package engine

import "time"

const ReadSchema = "station.engine.read/v1"

const (
	EngineRunning      = "running"
	EngineStopped      = "stopped"
	EngineNotInstalled = "not-installed"
)

type Reading struct {
	State string
	// value: the version the engine runs, and the one this station is.
	Version, Station string
	Up               time.Duration
	// value: the VMs the engine runs now.
	VMs int
	// value: until when the Awake session holds this Mac; nil when none does.
	AwakeUntil *time.Time
	// value: set while the engine is stopped by the owner: when, and by what.
	Stopped StopMark
}

func (r Reading) Skewed() bool {
	return r.State == EngineRunning && r.Version != "" && r.Station != "" && r.Version != r.Station
}

type StopMark struct {
	At time.Time `json:"at"`
	By string    `json:"by"`
}

type StopMarks interface {
	// value: the last stop the owner asked of this Mac's engine; ok is false
	// when none is kept.
	Read() (StopMark, bool)
	Write(m StopMark) error
}

type ReadingJSON struct {
	Schema string     `json:"schema"`
	Engine EngineJSON `json:"engine"`
}

type EngineJSON struct {
	State     string     `json:"state"`
	Version   *string    `json:"version"`
	UpS       *int64     `json:"up_s"`
	VMs       int        `json:"vms"`
	StoppedAt *time.Time `json:"stopped_at"`
	StoppedBy *string    `json:"stopped_by"`
}

func (r Reading) JSON() ReadingJSON {
	// invariant: SPEC §11.1: a version or an uptime the engine did not say
	// is null, never "" or 0.
	e := EngineJSON{State: r.State, VMs: r.VMs}
	if r.Version != "" {
		e.Version = &r.Version
	}
	if r.State == EngineRunning {
		up := int64(r.Up / time.Second)
		e.UpS = &up
	}
	if r.State == EngineStopped && !r.Stopped.At.IsZero() {
		at, by := r.Stopped.At.UTC(), r.Stopped.By
		e.StoppedAt, e.StoppedBy = &at, &by
	}
	return ReadingJSON{Schema: ReadSchema, Engine: e}
}
