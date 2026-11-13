package apps

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type UnknownError struct{ Name string }

func (e *UnknownError) Error() string { return "no app named \"" + e.Name + "\"" }

type ElsewhereError struct {
	App   string
	Kinds []catalog.MachineKind
	// value: a running VM where the app lives, named for the hint; empty
	// when none runs.
	VM string
}

func (e *ElsewhereError) Error() string { return e.App + " lives on " + LivesOn(e.Kinds) }

type NotInstalledError struct{ App string }

func (e *NotInstalledError) Error() string { return e.App + " is not installed" }

func LivesOn(kinds []catalog.MachineKind) string {
	// value: where an app lives, in the owner's words: a Mac is This Mac and
	// any other Mac, a VM a workspace.
	switch livesOn(kinds) {
	case "every-machine":
		return "every machine"
	case "mac-only":
		return "Macs only"
	case "vms-only":
		return "VMs only"
	case "linux-vms-only":
		return "Linux VMs only"
	case "macos-vms-only":
		return "macOS VMs only"
	}
	return "no machine"
}

func LivesOnToken(kinds []catalog.MachineKind) string {
	// value: where an app lives, as station.apps.list/v1's lives_on spells it.
	return livesOn(kinds)
}

func livesOn(livesOn []catalog.MachineKind) string {
	var mac, linux, macVM bool
	for _, p := range livesOn {
		switch p {
		case catalog.Mac:
			mac = true
		case catalog.Linux:
			linux = true
		case catalog.MacOS:
			macVM = true
		}
	}
	switch {
	case mac && linux:
		return "every-machine"
	case mac:
		return "mac-only"
	case linux && macVM:
		return "vms-only"
	case linux:
		return "linux-vms-only"
	case macVM:
		return "macos-vms-only"
	}
	return "no-machine"
}

type SignedOutError struct {
	// value: a store that waits on a sign-in it cannot ask for, read from
	// its silence: the App Store queues nothing while it is signed out.
	Store  string
	Waited time.Duration
	Hint   string
}

func (e *SignedOutError) Error() string {
	return e.Store + " did not answer in " + strconv.Itoa(int(e.Waited.Seconds())) + " s, as when it is signed out: " + e.Hint
}

// value: the item that provides a command is installed and the command is
// not there yet: it comes when that item is first opened.

type NotYetProvidedError struct{ Item, Command string }

func (e *NotYetProvidedError) Error() string {
	return e.Command + " did not come with " + e.Item + " yet"
}

type NotReversibleError struct{ App string }

func (e *NotReversibleError) Error() string { return e.App + " cannot be uninstalled by station" }

type UnrecordedError struct{ App string }

func (e *UnrecordedError) Error() string {
	return "station recorded adding none of " + e.App + "'s packages: they are yours"
}

type ManualInstallError struct{ App, Hint string }

func (e *ManualInstallError) Error() string { return e.App + " is installed by hand" }

type AppStoreError struct{ App string }

func (e *AppStoreError) Error() string { return e.App + " is the App Store's install" }

type NoUpgradePathError struct{ App string }

func (e *NoUpgradePathError) Error() string { return e.App + " has no upgrade path" }

// value: an app asked by a method it has none of on this machine, with the
// methods it has here and, for a move, the one that holds it.

type NoMethodError struct {
	App    string
	Method catalog.Method
	Has    []catalog.Method
	Held   catalog.Method
}

func (e *NoMethodError) Error() string { return e.App + " has no method " + string(e.Method) + " here" }

// value: an app asked at a version by a method whose install carries none
// to its manager.

type UnpinnableError struct {
	App     string
	Method  catalog.Method
	Version string
	Holds   string
}

func (e *UnpinnableError) Error() string {
	return string(e.Method) + " cannot hold " + e.App + " at " + e.Version
}
