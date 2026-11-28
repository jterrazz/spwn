package dotfile

import (
	"bytes"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/textblock"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

// invariant: a fragment, the file it goes into and the record are each
// measured before they are read, and once more as read: one past its
// bound is refused unread.
func (a *Applier) ours(from, refusal string) ([]byte, error) {
	info, err := fs.Stat(a.o.Ours, from)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxFragment {
		return nil, fmt.Errorf("the fragment %s is past 256 KiB: %s", from, refusal)
	}
	data, err := atMost(a.o.Ours.Open, from, maxFragment)
	if err == nil && len(data) > maxFragment {
		return nil, fmt.Errorf("the fragment %s is past 256 KiB: %s", from, refusal)
	}
	return data, err
}

func (a *Applier) fragment(c catalog.Copy) (*node, error) {
	data, err := a.ours(c.From, "none that large is merged")
	if err != nil {
		return nil, err
	}
	frag, err := parse(data)
	switch {
	case err != nil:
		return nil, fmt.Errorf("the fragment %s %w", c.From, err)
	case frag.shape != object:
		return nil, fmt.Errorf("the fragment %s holds no JSON object to merge", c.From)
	case len(leaves(frag, "")) == 0:
		return nil, fmt.Errorf("the fragment %s writes no value", c.From)
	}
	return frag, nil
}

// invariant: a file Machines merges into is the owner's, so one it cannot
// read whole as strict JSON, or past the bound, is refused and never
// rewritten.
func parseTarget(to string, data []byte) (*node, error) {
	if len(data) > maxTarget {
		return nil, fmt.Errorf("%s is past 1 MiB: Machines merges into no file that large", to)
	}
	if _, err := layoutOf(to, data); err != nil {
		return nil, err
	}
	tree, err := parse(data)
	switch {
	case err != nil:
		return nil, fmt.Errorf("%s %w: Machines writes nothing into it", to, err)
	case tree.shape != object:
		return nil, fmt.Errorf("%s holds no JSON object: Machines writes nothing into it", to)
	}
	return tree, nil
}

// value: what apply makes of the file's bytes: the merged text, what it
// wrote, and the owner's values it kept, the fragment's own first, then
// those of Machines' earlier values he changed since.
func mergeInto(to string, data []byte, there bool, frag *node, old []entry) (out []byte, w *merging, kept []string, err error) {
	tree := &node{shape: object}
	if there {
		if tree, err = parseTarget(to, data); err != nil {
			return nil, nil, nil, err
		}
	}
	w = newMerging(old)
	w.object(tree, frag, "")
	stale := takeBack(tree, w.stale())
	return encoded(tree, data, there), w, append(w.kept, stale...), nil
}

// value: the tree in the file's layout: its indent and line end, and its
// last line end when it had one; a file Machines makes ends with one.
func encoded(tree *node, data []byte, there bool) []byte {
	l, _ := layoutOf("", data)
	out := tree.encode(l)
	if !there || bytes.HasSuffix(data, []byte("\n")) {
		out = append(out, l.eol...)
	}
	return out
}

func (a *Applier) seeMerge(c catalog.Copy, old []entry) (seen, error) {
	frag, err := a.fragment(c)
	if err != nil {
		return seen{}, err
	}
	_, data, there, err := a.target(c.To, catalog.WriteMerge)
	if err != nil {
		return seen{}, err
	}
	after, _, _, err := mergeInto(c.To, data, there, frag, old)
	if err != nil {
		return seen{}, err
	}
	// invariant: the file is as Machines writes it when apply would write
	// nothing: a value of the owner's a merge keeps is his, not a drift.
	s := seen{same: there && bytes.Equal(after, data)}
	if !s.same {
		s.diff = settings.UnifiedDiff(string(after), string(data), "Machines' "+c.To, c.To)
	}
	if !there {
		return s, nil
	}
	tree, _ := parseTarget(c.To, data)
	material := []byte(c.To + "\x00")
	for _, l := range leaves(frag, "") {
		n, _, _ := tree.at(l.pointer)
		switch {
		case n == nil:
		case l.element:
			if n.shape == array && slices.ContainsFunc(n.items, l.val.equal) {
				s.there = true
				material = append(material, l.pointer+"\x00[]"+l.val.digest()+"\x00"...)
			}
		case l.val.shape != scalar && len(l.val.keys)+len(l.val.items) == 0 && n.shape == l.val.shape:
			// invariant: an empty object or array of the fragment is a
			// container Machines makes; what the owner puts in it is his,
			// so only its shape is Machines' to read, never his keys.
			s.there = true
			material = append(material, l.pointer+"\x00"+string(l.val.canonical())+"\x00"...)
		default:
			s.there = true
			material = append(append(append(material, l.pointer+"\x00"...), n.canonical()...), 0)
		}
	}
	s.there, s.material = s.there || s.same, material
	return s, nil
}

func (a *Applier) applyMerge(r *record, setting string, c catalog.Copy) ([]settings.ChangeStep, error) {
	frag, err := a.fragment(c)
	if err != nil {
		return nil, err
	}
	at, data, there, err := a.target(c.To, catalog.WriteMerge)
	if err != nil {
		return nil, err
	}
	w, _ := r.of(setting, c.To)
	out, m, kept, err := mergeInto(c.To, data, there, frag, w.Values)
	if err != nil {
		return nil, err
	}
	step := settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "as Machines wrote it: nothing written"}
	if !there || !bytes.Equal(out, data) {
		if err := a.land(&w, at, out, seenAs(data, there)); err != nil {
			return nil, err
		}
		step.Detail = "merged"
	}
	w.Values = m.now
	if len(w.Values) > 0 || w.Made {
		r.put(w)
	} else {
		r.drop(setting, c.To)
	}
	return append([]settings.ChangeStep{step}, keptSteps(c.To, kept)...), nil
}

func (a *Applier) undoMerge(r *record, setting string, c catalog.Copy) ([]settings.ChangeStep, error) {
	at, data, there, err := a.target(c.To, catalog.WriteMerge)
	if err != nil {
		return nil, err
	}
	w, held := r.of(setting, c.To)
	switch {
	case !there:
		r.drop(setting, c.To)
		return nil, nil
	case held < 0:
		return []settings.ChangeStep{{Label: c.To, State: catalog.StateOK, Detail: "no record of a value Machines wrote: left as it is"}}, nil
	}
	tree, err := parseTarget(c.To, data)
	if err != nil {
		return nil, err
	}
	kept := takeBack(tree, w.Values)
	step := settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "taken back"}
	switch out := encoded(tree, data, true); {
	case w.Made && len(tree.keys) == 0 && a.direct(c.To):
		if err := a.unmake(w, at, data); err != nil {
			return nil, err
		}
		step.Detail = "removed"
	case !bytes.Equal(out, data):
		if err := a.files.Swap(at, seenAs(data, true), out, 0o600); err != nil {
			return nil, err
		}
	default:
		step.Detail = "nothing of Machines' left"
	}
	r.drop(setting, c.To)
	return append([]settings.ChangeStep{step}, keptSteps(c.To, kept)...), nil
}

// value: one step per pointer kept, said once however many of Machines'
// entries met it.
func keptSteps(to string, pointers []string) []settings.ChangeStep {
	out := make([]settings.ChangeStep, 0, len(pointers))
	for i, p := range pointers {
		if slices.Contains(pointers[:i], p) {
			continue
		}
		out = append(out, settings.ChangeStep{Label: to + " " + p, State: catalog.StateDegraded, Detail: "yours, kept"})
	}
	return out
}

// invariant: a setting's block has its own namespace, `station setting:`,
// so it never meets a package's shell block (`station:<package>`), the
// guest's (`station hosts`) or the main block of one name.
func blockID(setting string) string { return "station setting:" + setting }

func (a *Applier) blockBody(c catalog.Copy) (string, error) {
	data, err := a.ours(c.From, "no block that large is written")
	if err != nil {
		return "", err
	}
	text := string(data)
	switch {
	case !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0:
		return "", fmt.Errorf("the fragment %s is not text", c.From)
	case strings.TrimSpace(text) == "":
		return "", fmt.Errorf("the fragment %s writes no line", c.From)
	}
	for _, line := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "# >>> ") || strings.HasPrefix(t, "# <<< ") {
			return "", fmt.Errorf("the fragment %s holds a marker line of its own", c.From)
		}
	}
	return strings.TrimSuffix(text, "\n") + "\n", nil
}

func textOf(to string, data []byte) (string, error) {
	switch {
	case len(data) > maxTarget:
		return "", fmt.Errorf("%s is past 1 MiB: Machines writes no block into a file that large", to)
	case !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0:
		return "", fmt.Errorf("%s is not a text file: Machines writes no block into it", to)
	case bytes.Contains(data, []byte("\r\n")):
		return "", fmt.Errorf("%s ends its lines with CRLF: Machines writes no block into it, and never converts them", to)
	}
	return string(data), nil
}

func (a *Applier) seeBlock(setting string, c catalog.Copy) (seen, error) {
	body, err := a.blockBody(c)
	if err != nil {
		return seen{}, err
	}
	_, data, _, err := a.target(c.To, catalog.WriteBlock)
	if err != nil {
		return seen{}, err
	}
	text, err := textOf(c.To, data)
	if err != nil {
		return seen{}, err
	}
	id := blockID(setting)
	have, state := textblock.Of(text, id)
	s := seen{same: state == textblock.Found && have == body}
	switch state {
	case textblock.Found:
		s.there, s.material = true, []byte(c.To+"\x00"+have)
	case textblock.Damaged:
		s.there, s.material = true, []byte(c.To+"\x00"+strings.Join(textblock.DamagedLines(text, id), "\x00"))
		return s, nil
	case textblock.Absent:
	}
	if after, changed := settings.UpsertPackageBlock(text, id, body); changed {
		s.diff = settings.UnifiedDiff(after, text, "Machines' "+c.To, c.To)
	}
	return s, nil
}

func (a *Applier) applyBlock(r *record, setting string, c catalog.Copy) ([]settings.ChangeStep, error) {
	body, err := a.blockBody(c)
	if err != nil {
		return nil, err
	}
	at, data, there, err := a.target(c.To, catalog.WriteBlock)
	if err != nil {
		return nil, err
	}
	text, err := textOf(c.To, data)
	if err != nil {
		return nil, err
	}
	id := blockID(setting)
	if _, state := textblock.Of(text, id); state == textblock.Damaged {
		return nil, fmt.Errorf("%s holds the block %s damaged (a start with no end, or the id twice): left as it is; delete it and apply again", c.To, id)
	}
	label := c.To + " [" + id + "]"
	w, _ := r.of(setting, c.To)
	after, changed := settings.UpsertPackageBlock(text, id, body)
	step := settings.ChangeStep{Label: label, State: catalog.StateOK, Detail: "as Machines wrote it: nothing written"}
	if changed {
		if err := a.land(&w, at, []byte(after), seenAs(data, there)); err != nil {
			return nil, err
		}
		step.Detail = "written"
	}
	w.Block = textDigest(body)
	r.put(w)
	return []settings.ChangeStep{step}, nil
}

func (a *Applier) undoBlock(r *record, setting string, c catalog.Copy) ([]settings.ChangeStep, error) {
	body, err := a.blockBody(c)
	if err != nil {
		return nil, err
	}
	at, data, there, err := a.target(c.To, catalog.WriteBlock)
	if err != nil || !there {
		r.drop(setting, c.To)
		return nil, err
	}
	text, err := textOf(c.To, data)
	if err != nil {
		return nil, err
	}
	id, label := blockID(setting), c.To+" ["+blockID(setting)+"]"
	w, held := r.of(setting, c.To)
	have, state := textblock.Of(text, id)
	switch {
	case held < 0 && state != textblock.Absent:
		return []settings.ChangeStep{{Label: label, State: catalog.StateOK, Detail: "no record of a block Machines wrote: left as it is"}}, nil
	case state == textblock.Absent:
		r.drop(setting, c.To)
		return nil, nil
	case state != textblock.Found || (have != body && textDigest(have) != w.Block):
		r.drop(setting, c.To)
		return []settings.ChangeStep{{Label: label, State: catalog.StateDegraded, Detail: "yours, kept"}}, nil
	}
	after, _ := textblock.Remove(text, id)
	step := settings.ChangeStep{Label: label, State: catalog.StateOK, Detail: "removed"}
	if w.Made && after == "" && a.direct(c.To) {
		err = a.unmake(w, at, data)
	} else {
		err = a.files.Swap(at, seenAs(data, true), []byte(after), 0o600)
	}
	if err != nil {
		return nil, err
	}
	r.drop(setting, c.To)
	return []settings.ChangeStep{step}, nil
}

func textDigest(text string) string { return digestOf([]byte(text)) }
