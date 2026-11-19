package background

import (
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Group string

const (
	GroupStation Group = "station"
	GroupKnown   Group = "known"
	GroupUnknown Group = "unknown"
	GroupOrphan  Group = "orphan"
)

type Run string

const (
	Running   Run = "running"
	Idle      Run = "idle"
	Failing   Run = "failing"
	NotLoaded Run = "not loaded"
	Unread    Run = "unknown"
)

type Task struct {
	Label     string
	Group     Group
	Run       Run
	PID       int
	Exit      int
	Scope     string
	Owner     string
	OwnerGone bool
	Plist     string
	Program   string
	Removal   string
	// value: the command that removes it by hand, whatever its state;
	// Removal is that command only where it is the task's fix.
	ByHand string
	State  catalog.State
}

func Judge(listed []Job, root, home string) []Task {
	// invariant: judging only reads what the job manager listed. A task keeps
	// the command that removes it by hand only where that is its fix: failing,
	// or an orphan, and never station's own. root and home turn a path into the
	// one the owner types.
	tasks := make([]Task, 0, len(listed))
	for _, l := range listed {
		j := Task{Label: l.Label, Scope: l.Scope, PID: l.PID, Plist: unrooted(l.File, root, home), Program: unrooted(l.Program, root, home), Removal: l.Removal, ByHand: l.Removal}
		switch {
		case l.Unread:
			j.Run = Unread
		case !l.Loaded:
			j.Run = NotLoaded
		case l.Running:
			j.Run = Running
		case l.LastExit != 0:
			j.Run, j.Exit = Failing, l.LastExit
		default:
			j.Run = Idle
		}
		livesOn(&j, l)
		tasks = append(tasks, j)
	}
	return tasks
}

func livesOn(j *Task, l Job) {
	owner, known := catalog.LaunchOwnerOf(j.Label)
	if known {
		j.Owner = owner.App
		j.OwnerGone = !l.OwnerExists
	}
	switch {
	case strings.HasPrefix(j.Label, catalog.StationLaunchPrefix):
		j.Group = GroupStation
	case j.Plist != "" && (!l.ProgramExists || j.OwnerGone):
		j.Group = GroupOrphan
		j.OwnerGone = true
	case known:
		j.Group = GroupKnown
	default:
		j.Group = GroupUnknown
	}
	switch {
	case j.Run == Failing:
		j.State = catalog.StateBroken
	case j.Group == GroupOrphan, j.Run == Unread:
		j.State = catalog.StateDegraded
	case j.Run == NotLoaded && j.Group == GroupStation:
		j.State = catalog.StateDegraded
	case j.Run == NotLoaded:
		j.State = catalog.StateAbsent
	default:
		j.State = catalog.StateOK
	}
	if j.Group == GroupStation || (j.Run != Failing && j.Group != GroupOrphan) {
		j.Removal = ""
	}
}

// value: the path the owner types: the system root dropped, the home as `~`.
func unrooted(p, root, home string) string {
	if r := path.Join("/", root); r != "/" {
		if rest, ok := strings.CutPrefix(p, r+"/"); ok {
			return "/" + rest
		}
	}
	if rest, ok := strings.CutPrefix(p, home+"/"); ok {
		return "~/" + rest
	}
	return p
}
