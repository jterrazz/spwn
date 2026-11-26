package contract

import (
	"context"
	"net/netip"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type Hypervisor interface {
	Name() string
	// invariant: Installed answers without running anything, so a Mac without
	// the hypervisor is told so before any call to it.
	Installed() bool
	Version(ctx context.Context) (string, error)
	Disks(ctx context.Context) ([]Disk, error)
	Fetch(ctx context.Context, img imagescontract.ImageLocation, p call.Progress) error
	Clone(ctx context.Context, from imagescontract.ImageLocation, vm string) error
	// invariant: Cloned is every VM this adapter cloned on this Mac and has
	// not deleted since, under the name a rename of its gave it, while the
	// disk under that name is still the one it cloned: a VM made or renamed
	// by hand, outside it, is never one, even under a name the adapter once
	// cloned.
	Cloned(ctx context.Context) ([]string, error)
	// invariant: Forget drops vm from what Cloned says, its disk already
	// gone: a delete that found no disk leaves no entry behind.
	Forget(ctx context.Context, vm string) error
	Resize(ctx context.Context, vm string, s machines.Size) error
	Rename(ctx context.Context, from, to string) error
	Boot(ctx context.Context, vm string, o BootOptions) (Run, error)
	Adopt(ctx context.Context, vm string) (Run, bool)
	Shutdown(ctx context.Context, vm string, grace time.Duration) error
	// invariant: Delete refuses a running VM; its caller stops the run first.
	Delete(ctx context.Context, vm string) error
	Discard(ctx context.Context, img imagescontract.ImageLocation) error
	Address(ctx context.Context, vm string, wait time.Duration) (netip.Addr, error)
	// invariant: Console is the last n meaningful lines of each of the VM's
	// two outputs, its serial console then the hypervisor's own, all of them
	// for n 0 or less; a VM that never ran has none.
	Console(ctx context.Context, vm string, n int) ([]string, error)
}

type DiskState string

const (
	DiskRunning   DiskState = "running"
	DiskStopped   DiskState = "stopped"
	DiskSuspended DiskState = "suspended"
	// value: a state the hypervisor said that station does not know: in
	// use, never read as stopped.
	DiskUnknown DiskState = "unknown"
)

type Disk struct {
	VM    string
	Bytes int64
	State DiskState
}

// value: how a VM boots: its network, the folders of its Mac it shares
// live, each under its tag (machines.MountTag), and how its disk takes
// its guest's flushes; the zero Sync is SyncFull.

type BootOptions struct {
	Network machines.Network
	Mounts  []machines.Mount
	Sync    machines.DiskSync
}

type Run interface {
	VM() string
	PID() int
	// invariant: Holder is the process that holds the VM's guest, whose CPU
	// and memory are the VM's on its Mac; 0 while none holds it yet.
	Holder() int
	// invariant: Ended fires once, when the run is over, with how it ended; a
	// run that outlives its engine is read again by Adopt after a restart.
	Ended() <-chan machines.RunEnd
}
