package main

import (
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
)

const callsPath = "apps/engine/remote/api/calls.json"

// invariant: the registry with zero Deps: only its Meta is read, no call runs.
func calls() []byte {
	registry, err := call.NewRegistry(
		accountscalls.All(accountscalls.Deps{}), activitycalls.All(activitycalls.Deps{}), appscalls.All(appscalls.Deps{}),
		awakecalls.All(awakecalls.Deps{}), backgroundcalls.All(backgroundcalls.Deps{}), diskcalls.All(diskcalls.Deps{}), enginecalls.All(enginecalls.Deps{}), agentscalls.All(agentscalls.Deps{}),
		imagescalls.All(imagescalls.Deps{}), machinefilecalls.All(machinefilecalls.Deps{}), machinescalls.All(machinescalls.Deps{}), overviewcalls.All(overviewcalls.Sections{}),
		portscalls.All(portscalls.Deps{}), settingscalls.All(settingscalls.Deps{}), setupcalls.All(setupcalls.Deps{}),
	)
	if err != nil {
		panic(err)
	}
	data, err := registry.Catalogue()
	if err != nil {
		panic(err)
	}
	return data
}
