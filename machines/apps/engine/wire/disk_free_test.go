package wire

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

// value: a disk on which a VM of the owner's lands in the old home just
// before the free sets it aside, after the free last read it.
type vmLanding struct {
	localtest.Disk
	landed bool
}

func (d *vmLanding) Move(ctx context.Context, from, to string) error {
	if !d.landed {
		d.landed = true
		_ = d.WriteFile("/h/.hv/vms/new-vm/disk.img", []byte("vm"), 0o600)
	}
	return d.Disk.Move(ctx, from, to)
}

func oldHome(t *testing.T) *localtest.FS {
	t.Helper()
	files := localtest.NewFS()
	_ = files.MkdirAll("/h/.hv", 0o755)
	_ = files.WriteFile("/h/.hv/cache/base.img", []byte("cache"), 0o600)
	return files
}

func shown(p string) string { return p }

func TestAnOldHomeAVMLandsInBeforeItIsSetAsideComesBackWholeNeverErased(t *testing.T) {
	files := oldHome(t)
	d := &vmLanding{Disk: localtest.Disk{FS: files}}
	err := eraseAside(t.Context(), d, files, "/h/.hv", []string{"/h/.hv/vms"}, shown)
	if err == nil || !strings.Contains(err.Error(), "holds VMs of yours (new-vm)") {
		t.Fatalf("the free answered %v, want the old home kept for the VM of the owner's in it", err)
	}
	for _, kept := range []string{"/h/.hv/vms/new-vm/disk.img", "/h/.hv/cache/base.img"} {
		if _, err := files.Stat(kept); err != nil {
			t.Errorf("%s is gone (%v), want the old home back whole", kept, err)
		}
	}
	if _, err := files.Stat("/h/.hv" + asideSuffix); err == nil {
		t.Error("the old home stays aside, want it back at its path")
	}
}

func TestAnOldHomeThatHoldsNoVMOnceSetAsideIsErased(t *testing.T) {
	files := oldHome(t)
	if err := eraseAside(t.Context(), localtest.Disk{FS: files}, files, "/h/.hv", []string{"/h/.hv/vms"}, shown); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"/h/.hv/cache/base.img", "/h/.hv" + asideSuffix + "/cache/base.img"} {
		if _, err := files.Stat(gone); err == nil {
			t.Errorf("%s stands after the free", gone)
		}
	}
}

func TestARecordThatTurnsUnreadableMidFreeStopsEveryOneAfterIt(t *testing.T) {
	reads := 0
	readable := func(context.Context) error {
		reads++
		if reads > 1 {
			return &disk.UnreadRecordError{Path: "~/.local/state/station/vms/api.json"}
		}
		return nil
	}
	var freed []string
	one := func(_ context.Context, f disk.Freeable) error {
		freed = append(freed, f.Name)
		return nil
	}
	frees := []disk.Freeable{{Kind: disk.FreeDisk, Name: "old-vm"}, {Kind: disk.FreeImage, Name: "linux@1"}}
	out := freeEach(t.Context(), frees, readable, one)
	if len(freed) != 1 || freed[0] != "old-vm" {
		t.Fatalf("freed %v, want old-vm alone: the record broke before the image", freed)
	}
	want := (&disk.UnreadRecordError{Path: "~/.local/state/station/vms/api.json"}).Error()
	if len(out) != 2 || out[1].Result != catalog.StateBroken || out[1].Reason != want {
		t.Fatalf("outcomes %+v, want the image broken by the unreadable record", out)
	}
}
