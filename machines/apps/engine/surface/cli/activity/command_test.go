package activitycmd

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

var errChannel = errors.New("ssh: unexpected packet in response to channel open: <nil>")

type silentGuest struct{ mini machine.Ref }

func (s silentGuest) Invoke(_ context.Context, target machine.Ref, _ string, in, out any, _ call.Progress) error {
	read, _ := in.(activitycalls.ReadIn)
	r, _ := out.(*activity.Report)
	switch {
	case target != s.mini:
		return errChannel
	case read.Run:
		*r = activity.Report{Source: activity.FromMacProcess, Last: activity.Reading{Run: &activity.Usage{CPU: 18, Memory: 15 << 30}}}
	default:
		*r = activity.Report{Source: activity.NotKept}
	}
	return nil
}

func (silentGuest) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("activity is read, not followed")
}

func TestAServerWhoseGuestStopsAnsweringBeforeItsMacNoticesIsShownFromItsMac(t *testing.T) {
	mini := machine.Ref{Name: "mac-mini"}
	env := kit.Env{Calls: silentGuest{mini: mini}, Self: machine.Ref{Name: "macbook"}}
	d := Deps{Label: func(string) string { return "mac-mini" }}
	v, err := vm(t.Context(), d, env, mini, "infra", machine.Server, true, read{})
	if err != nil {
		t.Fatalf("activity --on infra failed with %v; want infra's run as mac-mini measures it", err)
	}
	if v.Report.Source != activity.FromMacProcess || v.Report.Last.Run == nil || v.Report.Last.Run.Memory != 15<<30 {
		t.Errorf("activity --on infra shows %+v; want its run from mac-mini, 15 GB resident", v.Report)
	}
}

func TestActivityInsideAServerIsLabelledAsTheServerNotAsThisMac(t *testing.T) {
	self := machine.Ref{Name: "infra"}
	env := kit.Env{Calls: silentGuest{mini: machine.Ref{Name: "mac-mini"}}, Self: self, Kind: machine.Server}
	v, _ := mac(t.Context(), env, self, "infra", read{})
	if v.Kind != string(machine.Server) || v.Label != "infra" || !v.ThisMac {
		t.Errorf("activity inside a server is %q labeled %q; want the server, labeled infra", v.Kind, v.Label)
	}
}
