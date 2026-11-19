package machines

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type Sections struct {
	// value: whether the machine has each section of a machine's page, by
	// section.For on its kind; Images is the whole estate's, never one
	// machine's.
	Overview, Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake, SSH bool
}

func SectionsOf(vm VM) Sections {
	has := section.For(vm.Lifetime.Kind())
	in := func(id section.ID) bool { return slices.Contains(has, id) }
	return Sections{
		Overview: in(section.Overview), Activity: in(section.Activity), Apps: in(section.Apps), Settings: in(section.Settings),
		Disk: in(section.Disk), Ports: in(section.Ports), Accounts: in(section.Accounts), Background: in(section.Background),
		Awake: in(section.Awake), SSH: in(section.SSH),
	}
}
