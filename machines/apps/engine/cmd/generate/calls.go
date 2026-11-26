package main

import (
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	automationscalls "github.com/jterrazz/studio-station/apps/engine/automations/calls"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	skillscalls "github.com/jterrazz/studio-station/apps/engine/skills/calls"
	teamscalls "github.com/jterrazz/studio-station/apps/engine/teams/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

const callsPath = "apps/engine/remote/api/calls.json"

// invariant: the registry with zero Deps: only its Meta is read, no call runs.
func registry() *call.Registry {
	r, err := call.NewRegistry(
		agentscalls.All(agentscalls.Deps{}), skillscalls.All(skillscalls.Deps{}), projectcalls.All(projectcalls.Deps{}), teamscalls.All(teamscalls.Deps{}), automationscalls.All(automationscalls.Deps{}), accountscalls.All(accountscalls.Deps{}), activitycalls.All(activitycalls.Deps{}), appscalls.All(appscalls.Deps{}),
		awakecalls.All(awakecalls.Deps{}), backgroundcalls.All(backgroundcalls.Deps{}), diskcalls.All(diskcalls.Deps{}), enginecalls.All(enginecalls.Deps{}),
		imagescalls.All(imagescalls.Deps{}), journalcalls.All(journalcalls.Deps{}), machinefilecalls.All(machinefilecalls.Deps{}), machinescalls.All(machinescalls.Deps{}), overviewcalls.All(overviewcalls.Sections{}),
		portscalls.All(portscalls.Deps{}), settingscalls.All(settingscalls.Deps{}), setupcalls.All(setupcalls.Deps{}), vaultcalls.All(vaultcalls.Deps{}), grantscalls.All(grantscalls.Deps{}),
		devicescalls.All(devicescalls.Deps{}),
	)
	if err != nil {
		panic(err)
	}
	return r
}

func calls() []byte {
	owns := enginecalls.Owns()
	own := make([]call.Meta, 0, len(owns))
	for _, o := range owns {
		own = append(own, o.Meta)
	}
	data, err := registry().Catalogue(own...)
	if err != nil {
		panic(err)
	}
	return data
}
