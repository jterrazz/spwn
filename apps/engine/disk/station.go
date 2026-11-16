package disk

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/size"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type Holder string

const (
	HolderMachineDisks Holder = "machine-disks"
	HolderImages       Holder = "images"
	HolderLeftovers    Holder = "leftovers"
	HolderLogs         Holder = "logs"
	HolderThisMac      Holder = "this-mac"
	HolderFree         Holder = "free"
)

func (Holder) Enum() []string {
	holders := Holders()
	out := make([]string, len(holders))
	for i, h := range holders {
		out[i] = string(h)
	}
	return out
}

func Holders() []Holder {
	// value: what holds a Mac's disk, station's parts first, in the order
	// its bar draws them.
	return []Holder{HolderMachineDisks, HolderImages, HolderLeftovers, HolderLogs, HolderThisMac, HolderFree}
}

type Holding struct {
	Holder Holder
	Bytes  int64
}

type MachineDisk struct {
	Name  string
	Bytes int64
}

type Held struct {
	// value: the disks a Mac's hypervisor holds for station: each machine's,
	// each image version's, and what nothing uses: the image versions no
	// machine is made from and the leftovers.
	Machines []MachineDisk
	Images   []MachineDisk
	Unused   []Freeable
	// value: the hypervisor's old folders that hold VMs of the owner's,
	// never freed.
	Kept []Kept
	// value: the file of a machine record that cannot be read; while one
	// cannot, nothing is unused: its disk may be any of them.
	Unread string
}

type Kept struct {
	// value: an old folder of the hypervisor's that station never frees,
	// the owner's VMs it holds, and where they move.
	Name     string
	VMs      []string
	Sentence string
}

func KeptOf(folder, into string, vms []string) Kept {
	return Kept{
		Name: folder, VMs: vms,
		Sentence: folder + " holds VMs of yours (" + strings.Join(vms, ", ") + "), so station never frees it: move each into " + into + " first.",
	}
}

func TestVM(name string) bool {
	// value: a VM the machine tiers make with the hypervisor directly, under
	// these names; it is never a leftover.
	return strings.HasPrefix(name, "station-e2e-") || name == "station-play"
}

type FreeableKind string

const (
	FreeImage  FreeableKind = "image"
	FreeDisk   FreeableKind = "disk"
	FreeFolder FreeableKind = "folder"
)

func (FreeableKind) Enum() []string {
	return []string{string(FreeImage), string(FreeDisk), string(FreeFolder)}
}

type Freeable struct {
	// value: one thing `disk free-unused` deletes, under the holder whose
	// bytes it counts in, and what it is: an image version
	// (`station-linux@2026.10.20`), a disk station's hypervisor made that no
	// machine record names, or the hypervisor's old folder (`~/.tart`) once
	// it holds no VM.
	Holder Holder
	Kind   FreeableKind
	Name   string
	Bytes  int64
}

type HeldState int

const (
	// value: no hypervisor to ask: a machine that is not a Mac, or a Mac
	// with none installed.
	HeldNone HeldState = iota
	HeldPending
	// value: the hypervisor failed, or did not answer within its budget.
	HeldFailed
	HeldAnswered
)

type HeldReading struct {
	// value: what a Mac's hypervisor answered of the disks it holds; Held
	// is read only once State is HeldAnswered.
	State HeldState
	Held  Held
}

func (h Held) machines() int64 { return sumDisks(h.Machines) }

func (h Held) images() int64 { return sumDisks(h.Images) }

func sumDisks(disks []MachineDisk) int64 {
	var total int64
	for _, d := range disks {
		total += max(d.Bytes, 0)
	}
	return total
}

type StationPart struct {
	// value: what station holds on a Mac outside any machine's own files,
	// and the Mac itself beside it; each machine's size, the Mac first;
	// what nothing uses, and the gesture that frees it.
	Holds    []Holding
	Sentence string
	Itself   string
	Machines []MachineDisk
	CanFree  CanFree
	// value: the old folders station never frees, each with why.
	Kept    []Kept `json:",omitempty"`
	Verdict verdict.Verdict
}

func StationOf(total, free, logs int64, h Held, mac string) StationPart {
	// invariant: the Mac itself is what the volume uses beyond station's
	// parts, never below 0; the verdict sums exactly what CanFree frees.
	leftovers := h.unused(HolderLeftovers)
	held := h.machines() + h.images() + leftovers + max(logs, 0)
	itself := max(total-free-held, 0)
	machines := append([]MachineDisk{{Name: mac, Bytes: itself}}, h.Machines...)
	can := unusedFree(h.Unused)
	if h.Unread != "" {
		can = CanFree{Sentence: h.Unread + " cannot be read: nothing is offered to free until it can."}
	}
	return StationPart{
		Holds: []Holding{
			{Holder: HolderMachineDisks, Bytes: h.machines()},
			{Holder: HolderImages, Bytes: h.images()},
			{Holder: HolderLeftovers, Bytes: leftovers},
			{Holder: HolderLogs, Bytes: max(logs, 0)},
			{Holder: HolderThisMac, Bytes: itself},
			{Holder: HolderFree, Bytes: free},
		},
		Sentence: "Outside any machine, station holds " + size.Bytes(held) + ": machine disks, images, leftovers and its logs",
		Itself:   "This Mac itself holds " + size.Bytes(itself) + ".",
		Machines: machines,
		CanFree:  can,
		Kept:     h.Kept,
		Verdict:  freeable(can, h.Unread, mac),
	}
}

func unusedFree(frees []Freeable) CanFree {
	var bytes int64
	for _, f := range frees {
		bytes += max(f.Bytes, 0)
	}
	if len(frees) == 0 {
		return CanFree{Sentence: "Every image and every disk station holds is a machine's or the newest."}
	}
	return CanFree{
		Bytes:    bytes,
		Sentence: size.Bytes(bytes) + " can be freed: " + counted(frees) + ".",
		Call:     FreeUnusedCall,
		Frees:    frees,
	}
}

func counted(frees []Freeable) string {
	// value: what frees holds, counted by kind in the owner's words: "2
	// unused images and 1 leftover disk".
	words := []struct {
		kind      FreeableKind
		one, many string
	}{{FreeImage, "unused image", "unused images"}, {FreeDisk, "leftover disk", "leftover disks"}, {FreeFolder, "old folder", "old folders"}}
	var parts []string
	for _, w := range words {
		n := 0
		for _, f := range frees {
			if f.Kind == w.kind {
				n++
			}
		}
		switch {
		case n == 1:
			parts = append(parts, "1 "+w.one)
		case n > 1:
			parts = append(parts, strconv.Itoa(n)+" "+w.many)
		}
	}
	if len(parts) < 2 {
		return strings.Join(parts, "")
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func freeable(can CanFree, unread, mac string) verdict.Verdict {
	line := "on " + mac + ": no machine and no file of yours is touched"
	if unread != "" {
		return verdict.Verdict{Tone: verdict.Warn, Sentence: "A machine record cannot be read.", Line: line}
	}
	if can.Call != "" {
		return verdict.Verdict{Tone: verdict.Warn, Sentence: size.Bytes(can.Bytes) + " can be freed.", Line: line}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: "Nothing to free.", Line: line}
}

func (h Held) unused(holder Holder) int64 {
	var total int64
	for _, f := range h.Unused {
		if f.Holder == holder {
			total += max(f.Bytes, 0)
		}
	}
	return total
}

func (h Held) Holding() int64 { return h.machines() + h.images() + h.unused(HolderLeftovers) }
