package engine

import (
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	NoticeActivity = "activity"
	EventDiskLow   = "disk_low"
)

type Running struct {
	Name     string
	Lifetime model.Lifetime
	// value: the process that holds its guest on this Mac, 0 while none does.
	PID int
	// value: station's guest answers inside it.
	Guest bool
}
