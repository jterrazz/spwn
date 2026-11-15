package disk

import "strconv"

type Kind string

const (
	KindSystem   Kind = "system"
	KindApps     Kind = "apps"
	KindYourData Kind = "your-data"
	KindCaches   Kind = "caches"
	KindMachines Kind = "machines"
	KindFree     Kind = "free"
)

func (Kind) Enum() []string {
	kinds := Kinds()
	out := make([]string, len(kinds))
	for i, k := range kinds {
		out[i] = string(k)
	}
	return out
}

func Kinds() []Kind {
	// value: a machine's disk by kind, in the order its bar draws them.
	return []Kind{KindSystem, KindApps, KindYourData, KindCaches, KindMachines, KindFree}
}

type Amount struct {
	Kind  Kind
	Bytes int64
}

type CanFree struct {
	// value: what `disk free` frees here: the Can rebuild rows.
	Bytes    int64
	Sentence string
}

func KindsOf(total, free int64, s Survey, rows []Row, h HeldReading) ([]Amount, CanFree) {
	// invariant: a kind whose measure has not answered is absent, never a
	// guess: the system where the volume cannot tell it apart; machines and
	// images while the hypervisor is pending, failed or late, and on a
	// machine with none; your data, the remainder, whenever a kind it
	// subtracts is unknown.
	caches := Sum(Reading{Rows: rows}.Of(CanRebuild))
	var apps int64
	for _, t := range s.Rows {
		apps += max(t.Install, 0)
	}
	machines, held := int64(0), h.State == HeldAnswered
	if held {
		machines = h.Held.Holding()
	}
	var out []Amount
	known := s.System >= 0
	if known {
		out = append(out, Amount{Kind: KindSystem, Bytes: s.System})
	}
	out = append(out, Amount{Kind: KindApps, Bytes: apps})
	if known && (held || h.State == HeldNone) {
		out = append(out, Amount{Kind: KindYourData, Bytes: max(total-free-s.System-apps-caches-machines, 0)})
	}
	out = append(out, Amount{Kind: KindCaches, Bytes: caches})
	if held {
		out = append(out, Amount{Kind: KindMachines, Bytes: machines})
	}
	out = append(out, Amount{Kind: KindFree, Bytes: free})
	return out, canFree(caches)
}

func canFree(caches int64) CanFree {
	if caches <= 0 {
		return CanFree{Sentence: "No cache to free. What is left is yours or the system's."}
	}
	return CanFree{Bytes: caches, Sentence: "Caches hold " + Size(caches) + ". Each app rebuilds its own when it needs it."}
}

func Size(n int64) string {
	// invariant: decimal units and three figures at most, as every size
	// station prints reads (ui/atom.Bytes): 388 GB, 73.3 GB, 5.8 GB.
	const unit = 1000
	if n < unit {
		return strconv.FormatInt(max(n, 0), 10) + " B"
	}
	value, suffix := float64(n)/unit, 0
	for value >= unit-0.05 && suffix < len(sizeSuffixes)-1 {
		value /= unit
		suffix++
	}
	digits := 1
	if value >= 99.95 {
		digits = 0
	}
	return strconv.FormatFloat(value, 'f', digits, 64) + " " + string(sizeSuffixes[suffix]) + "B"
}

const sizeSuffixes = "KMGTPE"
