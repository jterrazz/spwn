package machines

import "time"

const Creating VMState = "creating"

type Birth struct {
	// value: how far its create is, per mille; the step it runs, in the
	// owner's words; when it reached that step.
	Progress int       `json:"progress"`
	Step     string    `json:"step"`
	At       time.Time `json:"at"`
}

// value: the create's steps in the owner's words, in order.
const (
	BornCloning = "cloning the image"
	BornBooting = "booting"
	BornAddress = "waiting for an address"
	BornSSH     = "waiting for SSH"
)

func BirthAt(step string, at time.Time) *Birth {
	progress := map[string]int{BornCloning: 50, BornBooting: 400, BornAddress: 600, BornSSH: 800}[step]
	return &Birth{Progress: progress, Step: step, At: at.UTC().Truncate(time.Second)}
}

const BirthStalls = 30 * time.Minute

const HalfMade = "was left half made"

const FreshFor = 8 * time.Second

func Fresh(vm VM, now time.Time) bool {
	return !vm.ReadyAt.IsZero() && !now.Before(vm.ReadyAt) && now.Sub(vm.ReadyAt) < FreshFor
}

func Born(vm VM, b *Birth, now time.Time) VM {
	// invariant: a birth wins over what the disk and the engine say, so a VM
	// reads creating until its create ends; one whose create stalled reads
	// stopped, half made, to start or delete.
	switch {
	case b == nil:
	case now.Sub(b.At) > BirthStalls:
		if vm.State == Running {
			return vm
		}
		vm.State, vm.Detail = Stopped, HalfMade
	default:
		vm.State, vm.Birth = Creating, b
	}
	return vm
}
