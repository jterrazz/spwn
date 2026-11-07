package engine

import (
	"slices"
	"strconv"

	model "github.com/jterrazz/studio-station/src/machines"
)

// value: the closed list of what the engine tells the owner of its
// servers (SPEC §10.4), each a Notification's Kind.
const (
	EventRestarting = "server_restarting"
	EventLoop       = "server_loop"
	EventPoweredOff = "server_powered_off"
	EventBack       = "server_back"
)

const EventsStream = "engine.events"

func Notifies(kind string) bool {
	// value: the closed list of SPEC §10.4 and §B3: the events the engine's events
	// watch streams, which the menu bar posts as its notifications.
	return slices.Contains([]string{EventRestarting, EventLoop, EventPoweredOff, EventBack, NoticeAwakeEnded, EventExposed, EventDiskLow,
		NoticeAgentWaiting, NoticeAgentFinished, NoticeVMReady, NoticeMacBack, NoticeShareEnded}, kind)
}

type Told struct {
	// value: an event of SPEC 10.4: its name, and the parts its text joins.
	Kind  string
	Parts []string
}

func Restarting(vm string, count int) Told {
	return Told{EventRestarting, []string{vm + " stopped and is restarting (↺ " + strconv.Itoa(count) + ")"}}
}

func Looping(vm string, r model.Restarts) Told {
	return Told{EventLoop, []string{vm + " is in a loop: " + strconv.Itoa(len(r.Recent)) + " restarts in " + strconv.Itoa(int(model.LoopWindow.Minutes())) + " min", "waiting " + strconv.Itoa(int(r.Wait.Seconds())) + " s"}}
}

func PoweredOff(vm string) Told {
	return Told{EventPoweredOff, []string{vm + " " + model.PoweredOffInside, "it stays stopped"}}
}

func Back(vm string) Told {
	return Told{EventBack, []string{vm + " is back", "all 4 health steps pass"}}
}
