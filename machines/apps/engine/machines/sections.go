package machines

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type Sections struct {
	// value: whether the machine has each section of a machine's page, by
	// section.For on its kind, but a VM whose create failed has its Overview
	// alone: it never booted, so no other section has anything to read;
	// Images is the whole estate's, never one machine's.
	Overview, Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake, SSH bool
}

func SectionsOf(vm VM) Sections {
	has := section.For(vm.Lifetime.Kind())
	in := func(id section.ID) bool { return slices.Contains(has, id) }
	if StandingOf(vm) == StandingNotCreated {
		return Sections{Overview: in(section.Overview)}
	}
	return Sections{
		Overview: in(section.Overview), Activity: in(section.Activity), Apps: in(section.Apps), Settings: in(section.Settings),
		Disk: in(section.Disk), Ports: in(section.Ports), Accounts: in(section.Accounts), Background: in(section.Background),
		Awake: in(section.Awake), SSH: in(section.SSH),
	}
}
