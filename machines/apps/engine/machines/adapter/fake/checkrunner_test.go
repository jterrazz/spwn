package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeChecksKeepTheCheckRunnerContract(t *testing.T) {
	contracttest.CheckRunner(t, func(*testing.T) contracttest.CheckRunnerSubject {
		w := &world.World{ThisMac: "mac-mini"}
		w.Mac("mac-mini").VMs = map[string]*world.VM{"infra": {PID: 2001}}
		return contracttest.CheckRunnerSubject{
			Runner:    CheckRunnerOf(w),
			Answering: func() machines.CheckTarget { return machines.CheckTarget{Dial: "192.168.64.5:6443", Name: "infra"} },
			Refusing: func() machines.CheckTarget {
				w.Script = append(w.Script, world.Fault{Call: "CheckRunner.Run"})
				return machines.CheckTarget{Dial: "192.168.64.5:6443", Name: "infra"}
			},
		}
	})
}
