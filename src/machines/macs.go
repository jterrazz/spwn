package machines

import "time"

type Added struct {
	Name, Label string
	// value: the station version the Mac answered with.
	Version string
	// value: what brings the two stations to one version, when they differ.
	Update string
}

type Back struct {
	// value: the Mac restarted, how long it took to answer again, and the
	// VMs it paused as they are now.
	Name   string
	Waited time.Duration
	VMs    []VM
}

type Unlocked struct {
	Name string
	// value: the Mac answered already: its disk was unlocked, nothing ran.
	Already bool
	// value: once unlocked it answered on the tailnet again in its budget.
	Answers bool
}

type Woken struct {
	Name string
	// value: the Mac answered already: nothing was sent.
	Already bool
	// value: a Wake-on-LAN packet was sent from this Mac; with Tell, the
	// owner is told once, the moment it answers again.
	Sent, Tell bool
}

type Reminded struct {
	Name string
	// value: the Mac answered already: nothing was armed.
	Already bool
}
