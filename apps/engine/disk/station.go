package disk

import "github.com/jterrazz/studio-station/apps/engine/core/verdict"

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
	// each image version's, and the bytes of the images nothing uses.
	Machines []MachineDisk
	Images   []MachineDisk
	Unused   int64
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
	// and the Mac itself beside it; each machine's size, the Mac first.
	Holds    []Holding
	Sentence string
	Machines []MachineDisk
	Verdict  verdict.Verdict
}

func StationOf(total, free, logs int64, h Held, mac string) StationPart {
	// invariant: leftovers hold nothing station can name yet, and the Mac
	// itself is what the volume uses beyond station's parts, never below 0.
	const leftovers = int64(0)
	held := h.machines() + h.images() + leftovers + max(logs, 0)
	itself := max(total-free-held, 0)
	machines := append([]MachineDisk{{Name: mac, Bytes: itself}}, h.Machines...)
	return StationPart{
		Holds: []Holding{
			{Holder: HolderMachineDisks, Bytes: h.machines()},
			{Holder: HolderImages, Bytes: h.images()},
			{Holder: HolderLeftovers, Bytes: leftovers},
			{Holder: HolderLogs, Bytes: max(logs, 0)},
			{Holder: HolderThisMac, Bytes: itself},
			{Holder: HolderFree, Bytes: free},
		},
		Sentence: "Outside any machine, station holds " + Size(held) + ": machine disks, images, leftovers and its logs",
		Machines: machines,
		Verdict:  freeable(h.Unused+leftovers, mac),
	}
}

func freeable(unused int64, mac string) verdict.Verdict {
	line := "on " + mac + ": no machine and no file of yours is touched"
	if unused > 0 {
		return verdict.Verdict{Tone: verdict.Warn, Sentence: Size(unused) + " can be freed.", Line: line}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: "Nothing to free.", Line: line}
}

func (h Held) Holding() int64 { return h.machines() + h.images() }
