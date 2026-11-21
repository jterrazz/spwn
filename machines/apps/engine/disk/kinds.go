package disk

import "github.com/jterrazz/studio-station/apps/engine/core/size"

type Kind string

const (
	KindSystem     Kind = "system"
	KindApps       Kind = "apps"
	KindYourData   Kind = "your-data"
	KindCaches     Kind = "caches"
	KindMachines   Kind = "machines"
	KindImageCache Kind = "image-cache"
	KindFree       Kind = "free"
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
	return []Kind{KindSystem, KindApps, KindYourData, KindCaches, KindMachines, KindImageCache, KindFree}
}

type Amount struct {
	Kind  Kind
	Bytes int64
}

const (
	// value: the calls a CanFree names, each the gesture that frees it.
	FreeCall       = "disk.free"
	FreeUnusedCall = "disk.free-unused"
)

type CanFree struct {
	// value: what a gesture frees here: the Can rebuild rows for `disk
	// free`, what nothing uses for `disk free-unused`.
	Bytes    int64
	Sentence string
	// value: the call that frees it; empty when there is nothing to free.
	Call string `json:",omitempty"`
	// value: the apps the call frees, by name; none frees what the call
	// frees bare.
	Apps  []string   `json:",omitempty"`
	Frees []Freeable `json:",omitempty"`
}

func KindsOf(total, free int64, s Survey, rows []Row, h HeldReading) ([]Amount, CanFree) {
	// invariant: a kind whose measure has not answered is absent, never 0:
	// the system where the volume cannot tell it apart or its walk pends,
	// apps and caches while a walk of theirs pends or past its budget,
	// machines and images until the hypervisor is heard (its last answer
	// stands while it is asked again, fails or is late), and on a machine
	// with none; your data, the remainder, whenever a kind it subtracts is
	// unknown. The image cache is the hypervisor's Can rebuild row, apart
	// from the caches and known with them; what can be freed is absent with
	// the caches.
	cacheRows := Reading{Rows: rows}.Of(CanRebuild)
	cachesKnown := !s.CachesUnknown
	var caches, imageCache int64
	for _, r := range cacheRows {
		cachesKnown = cachesKnown && r.Bytes >= 0
		if s.Hypervisor != "" && r.App == s.Hypervisor {
			imageCache += max(r.Bytes, 0)
		} else {
			caches += max(r.Bytes, 0)
		}
	}
	machines, held := int64(0), h.Heard()
	if held {
		machines = h.Held.Holding()
	}
	var out []Amount
	if s.System >= 0 {
		out = append(out, Amount{Kind: KindSystem, Bytes: s.System})
	}
	if s.Apps >= 0 {
		out = append(out, Amount{Kind: KindApps, Bytes: s.Apps})
	}
	if s.System >= 0 && s.Apps >= 0 && cachesKnown && (held || h.State == HeldNone) {
		out = append(out, Amount{Kind: KindYourData, Bytes: max(total-free-s.System-s.Apps-caches-imageCache-machines, 0)})
	}
	if cachesKnown {
		out = append(out, Amount{Kind: KindCaches, Bytes: caches})
	}
	if held {
		out = append(out, Amount{Kind: KindMachines, Bytes: machines})
	}
	if cachesKnown && s.Hypervisor != "" {
		out = append(out, Amount{Kind: KindImageCache, Bytes: imageCache})
	}
	out = append(out, Amount{Kind: KindFree, Bytes: free})
	if !cachesKnown {
		return out, CanFree{}
	}
	return out, canFree(caches)
}

func canFree(caches int64) CanFree {
	if caches <= 0 {
		return CanFree{Sentence: "No cache to free. What is left is yours or the system's."}
	}
	return CanFree{Bytes: caches, Sentence: "Caches hold " + size.Bytes(caches) + ". Each app rebuilds its own when it needs it.", Call: FreeCall}
}

func imageCacheOf(s Survey, rows []Row) (int64, CanFree) {
	// invariant: the image cache is freed only by its name, and says what
	// freeing it costs; it is -1 while its row is unsized, and on a machine
	// with no hypervisor it is 0 and offers nothing.
	if s.Hypervisor == "" {
		return 0, CanFree{}
	}
	var bytes int64
	for _, r := range rows {
		if r.Group != CanRebuild || r.App != s.Hypervisor {
			continue
		}
		if r.Bytes < 0 {
			return -1, CanFree{}
		}
		bytes += r.Bytes
	}
	if bytes <= 0 {
		return 0, CanFree{}
	}
	return bytes, CanFree{
		Bytes:    bytes,
		Sentence: "The image cache holds " + size.Bytes(bytes) + ": the upstream disks Machines downloaded. The next image built or fetched from one downloads it again.",
		Call:     FreeCall,
		Apps:     []string{s.Hypervisor},
	}
}
