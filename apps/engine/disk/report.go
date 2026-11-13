package disk

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Entry struct {
	Name     string
	Category string
	Disk     catalog.Disk
	Kinds    []catalog.MachineKind
}

type PathReport struct {
	Path  string
	Level catalog.Level
	Bytes int64
}

type MisplacedReport struct {
	Name  string
	Path  string
	Bytes int64
	// value: where its move takes it, as the owner reads it.
	To     string
	Repair string
}

type Tally struct {
	Entry     Entry
	Installed bool
	Unknown   bool
	Within    time.Duration
	State     catalog.State
	Bytes     map[catalog.Level]int64
	Install   int64
	Paths     []PathReport
	Misplaced []MisplacedReport
	Pending   bool
	// value: a pending size the walk has never measured, so the row cannot
	// know yet which groups it will fill.
	Unmeasured bool
}

func (r Tally) Total() int64 {
	total := r.Install
	for _, b := range r.Bytes {
		total += b
	}
	for _, s := range r.Misplaced {
		total += s.Bytes
	}
	return total
}

type Survey struct {
	Rows      []Tally
	Misplaced []MisplacedReport
	Other     int64
}

type Outcome struct {
	Name   string
	Result catalog.State
	Detail string
	Reason string
	Hint   string
	Freed  int64
	Erase  bool
}

func Failures(outcomes []Outcome) (failed int, firstFailed Outcome) {
	for _, o := range outcomes {
		if o.Result != catalog.StateBroken {
			continue
		}
		if failed == 0 {
			firstFailed = o
		}
		failed++
	}
	return failed, firstFailed
}

const (
	StationName     = "station"
	StationCategory = "Station"
)
