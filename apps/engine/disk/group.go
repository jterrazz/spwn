package disk

import (
	"cmp"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Group string

const (
	Misplaced    Group = "misplaced"
	CanRebuild   Group = "can-rebuild"
	YourData     Group = "your-data"
	NeverTouched Group = "never-touched"
)

func (Group) Enum() []string {
	groups := Groups()
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = string(g)
	}
	return out
}

func Groups() []Group {
	// value: the Disk groups in the order they are read: what is misplaced
	// first, then by what an erase would lose.
	return []Group{Misplaced, CanRebuild, YourData, NeverTouched}
}

type Row struct {
	App    string
	Group  Group
	Paths  []string
	Bytes  int64
	MoveTo string
	// value: the walk has not answered: Bytes is the last size known.
	Pending bool
}

type Reading struct {
	Total, Free int64
	Rows        []Row
	Pending     bool
}

func (r Reading) Of(g Group) []Row {
	var out []Row
	for _, row := range r.Rows {
		if row.Group == g {
			out = append(out, row)
		}
	}
	return out
}

func Sum(rows []Row) int64 {
	var total int64
	for _, r := range rows {
		total += max(r.Bytes, 0)
	}
	return total
}

func GroupOf(l catalog.Level) Group {
	// value: what each catalogue level risks if erased: a cache nothing,
	// data the owner's work, a key or a setting is never station's to erase.
	switch l {
	case catalog.LevelCache:
		return CanRebuild
	case catalog.LevelData:
		return YourData
	case catalog.LevelSettings, catalog.LevelKept:
	}
	return NeverTouched
}

func Classify(s Survey) []Row {
	// invariant: one row per app and group it holds bytes in, sorted by size
	// within its group; a stray is Misplaced whatever it holds; station's own
	// data are its VMs and images, which machines and images own, never a
	// row. A pending row keeps its last-known groups, so no row leaves when
	// the walk answers; one never measured, or not sized, waits in Can
	// rebuild when its app has a clean command.
	var out []Row
	for _, t := range s.Rows {
		for _, st := range t.Misplaced {
			out = append(out, Row{App: t.Entry.Name, Group: Misplaced, Paths: []string{st.Path}, Bytes: st.Bytes, MoveTo: st.To, Pending: t.Pending})
		}
		frees := len(t.Entry.Disk.Free) > 0
		if t.Unknown {
			if frees {
				out = append(out, Row{App: t.Entry.Name, Group: CanRebuild, Bytes: -1})
			}
			continue
		}
		bytes := map[Group]int64{}
		for level, b := range t.Bytes {
			if g := GroupOf(level); g != YourData || t.Entry.Name != Station {
				bytes[g] += max(b, 0)
			}
		}
		paths := map[Group][]string{}
		for _, p := range t.Paths {
			if p.Bytes > 0 {
				paths[GroupOf(p.Level)] = append(paths[GroupOf(p.Level)], p.Path)
			}
		}
		for _, g := range Groups()[1:] {
			if bytes[g] > 0 || (t.Unmeasured && frees && g == CanRebuild) {
				out = append(out, Row{App: t.Entry.Name, Group: g, Paths: append([]string{}, paths[g]...), Bytes: bytes[g], Pending: t.Pending})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Row) int {
		return cmp.Or(cmp.Compare(slices.Index(Groups(), a.Group), slices.Index(Groups(), b.Group)), cmp.Compare(b.Bytes, a.Bytes), cmp.Compare(a.App, b.App))
	})
	return out
}

const Station = "station"
