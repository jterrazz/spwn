package automations

import (
	"errors"
	"time"
)

const (
	Cron = "cron"
	FS   = "fs"
)

type Trigger struct {
	// value: Cron or FS, the one trigger the project declares.
	Kind string
	// value: the cron expression, or the path watched, as declared.
	Spec string
}

type Automation struct {
	Name    string
	Agent   string
	Trigger Trigger
	// value: when the newest run of this automation started; zero when
	// none ran.
	LastFired time.Time
	Fires     int
	Failed    int
}

type Run struct {
	Name string
	// value: the cron slot the run covers; zero for a file trigger.
	Scheduled time.Time
	At        time.Time
	OK        bool
	Error     string
	Output    string
}

var ErrNotFound = errors.New("no automation of that name")

func Reserved(name string) bool {
	// invariant: `automations log` is the command that reads the runs, so
	// an automation of that name is listed and has no detail to ask for.
	return name == "log"
}
