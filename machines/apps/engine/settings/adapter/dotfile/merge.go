package dotfile

import (
	"cmp"
	"slices"
	"strings"
)

// invariant: a merge writes into the owner's object key by key: a key he
// does not have is added, an object or an array he has is merged into,
// an element goes into an array only when no deep-equal one is there, and
// a value of his is never replaced; a value Machines wrote and he left as
// it is is Machines' own, written anew when the fragment moved, whatever
// shape it moved to.

type merging struct {
	old  []entry
	used []bool
	now  []entry
	// value: the pointers of the owner's values the fragment says
	// otherwise, left as they are.
	kept []string
}

func newMerging(old []entry) *merging {
	return &merging{old: old, used: make([]bool, len(old))}
}

func (w *merging) take(e entry) bool {
	for i, o := range w.old {
		if !w.used[i] && o == e {
			w.used[i] = true
			return true
		}
	}
	return false
}

func (w *merging) carry(e entry) {
	if w.take(e) {
		w.now = append(w.now, e)
	}
}

func (w *merging) stale() []entry {
	var out []entry
	for i, o := range w.old {
		if !w.used[i] {
			out = append(out, o)
		}
	}
	return out
}

func (w *merging) object(have, frag *node, at string) {
	for _, m := range frag.keys {
		p := child(at, m.name)
		i := have.member(m.name)
		if i < 0 {
			have.keys = append(have.keys, member{name: m.name, raw: m.raw, val: w.made(m.val, p)})
			continue
		}
		mine := have.keys[i].val
		switch {
		case mine.shape == object && m.val.shape == object:
			w.carry(entry{Pointer: p, Kind: kindObject})
			w.object(mine, m.val, p)
		case mine.shape == array && m.val.shape == array:
			w.carry(entry{Pointer: p, Kind: kindArray})
			w.array(mine, m.val, p)
		case mine.equal(m.val):
			w.carry(entry{Pointer: p, Kind: kindValue, Digest: mine.digest()})
		default:
			taken, ours := w.owned(mine, p, nil)
			if !ours {
				w.kept = append(w.kept, p)
				continue
			}
			for _, o := range taken {
				w.used[o] = true
			}
			have.keys[i].val = w.made(m.val, p)
		}
	}
}

// invariant: what sits at a key is Machines' alone when every part of it
// is an entry of the record not used yet: a value by its digest, an object
// or an array it made, each member and element of it so: only then does a
// fragment of another value or shape replace it, in the same pass; one
// part of the owner's leaves the whole his.
func (w *merging) owned(n *node, at string, taken []int) ([]int, bool) {
	switch n.shape {
	case scalar:
		return w.claim(entry{Pointer: at, Kind: kindValue, Digest: n.digest()}, taken)
	case object:
		var ok bool
		taken, ok = w.claim(entry{Pointer: at, Kind: kindObject}, taken)
		for _, m := range n.keys {
			if !ok {
				break
			}
			taken, ok = w.owned(m.val, child(at, m.name), taken)
		}
		return taken, ok
	case array:
	}
	var ok bool
	taken, ok = w.claim(entry{Pointer: at, Kind: kindArray}, taken)
	for _, it := range n.items {
		if !ok {
			break
		}
		taken, ok = w.claim(entry{Pointer: at, Kind: kindElement, Digest: it.digest()}, taken)
	}
	return taken, ok
}

func (w *merging) claim(e entry, taken []int) ([]int, bool) {
	for i, o := range w.old {
		if !w.used[i] && o == e && !slices.Contains(taken, i) {
			return append(taken, i), true
		}
	}
	return taken, false
}

func (w *merging) array(have, frag *node, at string) {
	for _, e := range frag.items {
		d := entry{Pointer: at, Kind: kindElement, Digest: e.digest()}
		if slices.ContainsFunc(have.items, e.equal) {
			w.carry(d)
			continue
		}
		have.items = append(have.items, e.clone())
		w.take(d)
		w.now = append(w.now, d)
	}
}

func (w *merging) made(frag *node, at string) *node {
	switch frag.shape {
	case object:
		n := &node{shape: object}
		w.now = append(w.now, entry{Pointer: at, Kind: kindObject})
		w.object(n, frag, at)
		return n
	case array:
		n := &node{shape: array}
		w.now = append(w.now, entry{Pointer: at, Kind: kindArray})
		w.array(n, frag, at)
		return n
	case scalar:
	}
	w.now = append(w.now, entry{Pointer: at, Kind: kindValue, Digest: frag.digest()})
	return frag.clone()
}

// invariant: taking back removes only what the entries name and the file
// still holds exactly: a value whose digest is the one written, the first
// element deep-equal to the one added, then each object or array Machines
// made, deepest first, once it is empty and still of the shape it made. A
// value changed since, or of another shape, is left, and its pointer said.
func takeBack(root *node, entries []entry) (kept []string) {
	entries = slices.Clone(entries)
	slices.SortStableFunc(entries, func(a, b entry) int {
		return cmp.Compare(rank(b), rank(a))
	})
	for _, e := range entries {
		n, parent, i := root.at(e.Pointer)
		if n == nil || parent == nil {
			continue
		}
		switch e.Kind {
		case kindValue:
			if n.digest() != e.Digest {
				kept = append(kept, e.Pointer)
				continue
			}
			parent.keys = slices.Delete(parent.keys, i, i+1)
		case kindElement:
			if n.shape != array {
				kept = append(kept, e.Pointer)
				continue
			}
			if j := slices.IndexFunc(n.items, func(it *node) bool { return it.digest() == e.Digest }); j >= 0 {
				n.items = slices.Delete(n.items, j, j+1)
			}
		case kindObject, kindArray:
			switch {
			case n.shape != e.Kind.shape():
				kept = append(kept, e.Pointer)
			case len(n.keys)+len(n.items) == 0:
				parent.keys = slices.Delete(parent.keys, i, i+1)
			}
		}
	}
	return kept
}

// value: the shape of the container an entry says Machines made.
func (k kind) shape() shape {
	if k == kindObject {
		return object
	}
	return array
}

// value: values and elements first, then the containers, deepest first.
func rank(e entry) int {
	if e.Kind == kindValue || e.Kind == kindElement {
		return 1 << 20
	}
	return strings.Count(e.Pointer, "/")
}

// value: one place a fragment writes: a value at a key, an element of an
// array, or an empty object or array.

type leaf struct {
	pointer string
	element bool
	val     *node
}

func leaves(frag *node, at string) []leaf {
	var out []leaf
	for _, m := range frag.keys {
		p := child(at, m.name)
		switch {
		case m.val.shape == object && len(m.val.keys) > 0:
			out = append(out, leaves(m.val, p)...)
		case m.val.shape == array && len(m.val.items) > 0:
			for _, it := range m.val.items {
				out = append(out, leaf{pointer: p, element: true, val: it})
			}
		default:
			out = append(out, leaf{pointer: p, val: m.val})
		}
	}
	return out
}
