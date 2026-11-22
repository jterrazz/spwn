package disk

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/size"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type MachineDisk struct {
	Name  string
	Bytes int64
}

type Held struct {
	// value: every disk a Mac's hypervisor holds in Machines' home: each
	// machine's, each image version's, every other VM's (a test VM, one made
	// by hand, one no record names that is no leftover), and what nothing
	// uses: the image versions no machine is made from and the leftovers.
	Machines []MachineDisk
	Images   []MachineDisk
	Others   []MachineDisk
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
		Sentence: folder + " holds VMs of yours (" + strings.Join(vms, ", ") + "), so Machines never frees it: move each into " + into + " first.",
	}
}

func TestVM(name string) bool {
	// value: a VM the machine tiers make with the hypervisor directly, under
	// these names; it is never a leftover, and its disk is another VM's.
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
	// (`base-ubuntu@2026.10.20`), a disk station's hypervisor made that no
	// machine record names, or the hypervisor's old folder (`~/.tart`) once
	// it holds no VM.
	Holder Kind
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
	// is read only once State is HeldAnswered, or while Known.
	State HeldState
	Held  Held
	// value: Held is the last answer the hypervisor gave this engine,
	// standing while it is asked again, fails or answers late.
	Known bool
}

func (h HeldReading) Heard() bool { return h.State == HeldAnswered || h.Known }

func (h Held) machines() int64 { return sumDisks(h.Machines) }

func (h Held) images() int64 { return sumDisks(h.Images) }

func (h Held) others() int64 { return sumDisks(h.Others) }

func sumDisks(disks []MachineDisk) int64 {
	var total int64
	for _, d := range disks {
		total += max(d.Bytes, 0)
	}
	return total
}

type StationPart struct {
	// value: what the Mac itself holds and what Machines holds on it
	// outside any machine's own files, said; each machine's size, the Mac
	// first; what nothing uses, and the gesture that frees it. Its parts
	// are the reading's Kinds.
	Sentence string
	Machines []MachineDisk
	CanFree  CanFree
	// value: the old folders station never frees, each with why.
	Kept    []Kept `json:",omitempty"`
	Verdict verdict.Verdict
}

func StationOf(total, free, logs, imageCache int64, h Held, mac string) StationPart {
	// invariant: the Mac itself is what the volume uses beyond station's
	// parts, never below 0; the verdict sums exactly what CanFree frees.
	held := h.Holding() + max(imageCache, 0) + max(logs, 0)
	itself := max(total-free-held, 0)
	machines := append([]MachineDisk{{Name: mac, Bytes: itself}}, h.Machines...)
	can := unusedFree(h.Unused)
	if h.Unread != "" {
		can = CanFree{Sentence: h.Unread + " cannot be read: nothing is offered to free until it can."}
	}
	return StationPart{
		Sentence: "This Mac itself holds " + size.Bytes(itself) + ". Outside any machine, Machines holds " + size.Bytes(held) + ".",
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
		return CanFree{Sentence: "Every image and every disk Machines holds is a machine's or the newest."}
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

func (h Held) unused(holder Kind) int64 {
	var total int64
	for _, f := range h.Unused {
		if f.Holder == holder {
			total += max(f.Bytes, 0)
		}
	}
	return total
}

func (h Held) Holding() int64 {
	return h.machines() + h.images() + h.others() + h.unused(KindLeftovers)
}
