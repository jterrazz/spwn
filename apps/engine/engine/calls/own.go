package calls

import (
	"reflect"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Own struct {
	// value: a call or a watch the engine serves beside the registry, as the
	// wire carries it.
	call.Meta
	// value: what a watch answers first, before its events; nil for a watch
	// whose every answer is its Out.
	Opens reflect.Type
	// value: the `event` of each engine.Event a watch streams after its
	// opening, in the order the stream's schema lists them.
	Events []string
}

func Owns() []Own {
	// invariant: what the engine serves beside the registry; the engine
	// process's server binds the same names to the same types, which its own
	// test holds.
	none := reflect.TypeFor[struct{}]()
	ok := reflect.TypeFor[struct{}]()
	out := []Own{
		served(engine.CallHello, none, reflect.TypeFor[engine.Hello]()),
		served(engine.CallVMs, none, reflect.TypeFor[[]engine.VMView]()),
		served(engine.CallPorts, reflect.TypeFor[engine.VMIn](), reflect.TypeFor[ports.MacPorts]()),
		served(engine.CallReserve, reflect.TypeFor[engine.ReserveIn](), reflect.TypeFor[ports.SharedPort]()),
		served(engine.CallRelease, reflect.TypeFor[engine.ReserveIn](), reflect.TypeFor[ports.Reservation]()),
		served(engine.CallGuestExec, reflect.TypeFor[engine.ExecIn](), reflect.TypeFor[engine.ExecOut]()),
		served(engine.CallAccounts, none, reflect.TypeFor[engine.AccountsView]()),
		served(engine.CallStopAll, none, reflect.TypeFor[engine.StopAll]()),
		served(engine.CallAwakeView, none, reflect.TypeFor[awake.View]()),
		served(engine.CallKeepAwake, reflect.TypeFor[engine.AwakeIn](), reflect.TypeFor[awake.Awoken]()),
		served(engine.CallAwakeOff, none, reflect.TypeFor[engine.AwakeOff]()),
		served(engine.CallActivity, none, reflect.TypeFor[engine.ActivityOut]()),
		served(engine.CallKeepSettings, reflect.TypeFor[engine.KeepSettingsIn](), ok),
		served(engine.CallLastSettings, reflect.TypeFor[engine.VMIn](), reflect.TypeFor[engine.LastSettingsOut]()),
		served(engine.CallRefresh, reflect.TypeFor[engine.PagesIn](), ok),
		served(engine.CallAgentsHeld, none, reflect.TypeFor[agents.Held]()),
	}
	for _, verb := range engine.VMVerbs() {
		out = append(out, served(engine.CallVM(verb), reflect.TypeFor[engine.VMIn](), reflect.TypeFor[*struct{}]()))
	}
	opened, event := reflect.TypeFor[engine.Opened](), reflect.TypeFor[engine.Event]()
	return append(out,
		watched(engine.CallPage, reflect.TypeFor[engine.PagesIn](), opened, event, "page", engine.EventSample, "tick", "notice"),
		watched(engine.CallEvents, none, opened, event, "notice"),
		watched(engine.CallWatch, reflect.TypeFor[engine.WatchIn](), nil, reflect.TypeFor[engine.Watching]()),
	)
}

func served(name string, in, out reflect.Type) Own {
	return Own{Meta: call.Meta{Name: name, Where: call.InEngine, In: in.Name(), Out: out.Name(), InType: in, OutType: out}}
}

func watched(name string, in, opens, out reflect.Type, events ...string) Own {
	o := served(name, in, out)
	o.Watch, o.Opens, o.Events = true, opens, events
	if opens != nil {
		o.Out = opens.Name() + "|" + out.Name()
	}
	return o
}
