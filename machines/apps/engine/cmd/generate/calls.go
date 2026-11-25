package main

import (
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
	wikicalls "github.com/jterrazz/studio-station/apps/engine/wiki/calls"
)

const callsPath = "apps/engine/remote/api/calls.json"

// invariant: the registry with zero Deps: only its Meta is read, no call runs.
func registry() *call.Registry {
	r, err := call.NewRegistry(
		accountscalls.All(accountscalls.Deps{}), activitycalls.All(activitycalls.Deps{}), appscalls.All(appscalls.Deps{}),
		awakecalls.All(awakecalls.Deps{}), backgroundcalls.All(backgroundcalls.Deps{}), diskcalls.All(diskcalls.Deps{}), enginecalls.All(enginecalls.Deps{}),
		imagescalls.All(imagescalls.Deps{}), journalcalls.All(journalcalls.Deps{}), machinefilecalls.All(machinefilecalls.Deps{}), machinescalls.All(machinescalls.Deps{}), overviewcalls.All(overviewcalls.Sections{}),
		portscalls.All(portscalls.Deps{}), settingscalls.All(settingscalls.Deps{}), setupcalls.All(setupcalls.Deps{}), vaultcalls.All(vaultcalls.Deps{}), grantscalls.All(grantscalls.Deps{}), wikicalls.All(wikicalls.Deps{}),
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
