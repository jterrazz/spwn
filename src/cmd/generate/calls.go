package main

import (
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/src/background/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"
)

const callsPath = "src/remote/api/calls.json"

// invariant: the registry with zero Deps: only its Meta is read, no call runs.
func calls() []byte {
	registry, err := call.NewRegistry(
		accountscalls.All(accountscalls.Deps{}), activitycalls.All(activitycalls.Deps{}), appscalls.All(appscalls.Deps{}),
		awakecalls.All(awakecalls.Deps{}), backgroundcalls.All(backgroundcalls.Deps{}), diskcalls.All(diskcalls.Deps{}), enginecalls.All(enginecalls.Deps{}), agentscalls.All(agentscalls.Deps{}),
		imagescalls.All(imagescalls.Deps{}), machinescalls.All(machinescalls.Deps{}), overviewcalls.All(overviewcalls.Sections{}),
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
