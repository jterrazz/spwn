package dotfile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// value: the bounds a fragment holds to: the file it writes into, the
// fragment itself, and how deep either nests.

const (
	maxTarget   = 1 << 20
	maxFragment = 256 << 10
	maxDepth    = 32
)

type shape int

const (
	scalar shape = iota
	object
	array
)

// value: one JSON value as its file spells it: an object's members in the
// file's order, each key and scalar by its own bytes, so what station
// writes back keeps the owner's order, escapes and numbers.

type node struct {
	shape shape
	raw   []byte
	keys  []member
	items []*node
}

type member struct {
	name string
	raw  []byte
	val  *node
}

var (
	errNotStrict = errors.New("is not strict JSON (a comment, a trailing comma, a key twice)")
	errTooDeep   = errors.New("nests deeper than " + strconv.Itoa(maxDepth))
)

func parse(data []byte) (*node, error) {
	// invariant: strict JSON or nothing: a comment, a trailing comma, a key
	// twice or a second value is refused, never read around.
	if !json.Valid(data) {
		return nil, errNotStrict
	}
	p := parser{dec: json.NewDecoder(bytes.NewReader(data)), data: data}
	p.dec.UseNumber()
	n, err := p.value(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errNotStrict
	}
	return n, nil
}

type parser struct {
	dec  *json.Decoder
	data []byte
}

func (p *parser) token() (json.Token, []byte, error) {
	from := p.dec.InputOffset()
	tok, err := p.dec.Token()
	if err != nil {
		return nil, nil, err
	}
	return tok, bytes.TrimLeft(p.data[from:p.dec.InputOffset()], " \t\r\n,:"), nil
}

func (p *parser) value(depth int) (*node, error) {
	tok, raw, err := p.token()
	if err != nil {
		return nil, err
	}
	return p.from(tok, raw, depth)
}

func (p *parser) from(tok json.Token, raw []byte, depth int) (*node, error) {
	d, delim := tok.(json.Delim)
	if !delim {
		return &node{shape: scalar, raw: slices.Clone(raw)}, nil
	}
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	if d == '[' {
		n := &node{shape: array}
		for p.dec.More() {
			item, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			n.items = append(n.items, item)
		}
		_, err := p.dec.Token()
		return n, err
	}
	n := &node{shape: object}
	for p.dec.More() {
		tok, raw, err := p.token()
		if err != nil {
			return nil, err
		}
		name, _ := tok.(string)
		if n.member(name) >= 0 {
			return nil, errNotStrict
		}
		val, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		n.keys = append(n.keys, member{name: name, raw: slices.Clone(raw), val: val})
	}
	_, err := p.dec.Token()
	return n, err
}

func (n *node) member(name string) int {
	return slices.IndexFunc(n.keys, func(m member) bool { return m.name == name })
}

func (n *node) clone() *node {
	c := &node{shape: n.shape, raw: n.raw}
	for _, m := range n.keys {
		c.keys = append(c.keys, member{name: m.name, raw: m.raw, val: m.val.clone()})
	}
	for _, it := range n.items {
		c.items = append(c.items, it.clone())
	}
	return c
}

// value: what a file's layout keeps of the file: its indent, and its line
// end, `\n` or `\r\n`.
type layout struct{ indent, eol string }

// value: the file's text: JSON.stringify's layout under the file's own
// indent and line end, every key and scalar by its bytes.
func (n *node) encode(l layout) []byte {
	var b bytes.Buffer
	n.write(&b, l, 0)
	return b.Bytes()
}

func (n *node) write(b *bytes.Buffer, indent layout, depth int) {
	switch {
	case n.shape == scalar:
		b.Write(n.raw)
		return
	case n.shape == object && len(n.keys) == 0:
		b.WriteString("{}")
		return
	case n.shape == array && len(n.items) == 0:
		b.WriteString("[]")
		return
	}
	open, end := byte('{'), byte('}')
	if n.shape == array {
		open, end = '[', ']'
	}
	b.WriteByte(open)
	for i := range max(len(n.keys), len(n.items)) {
		if i > 0 {
			b.WriteByte(',')
		}
		newline(b, indent, depth+1)
		if n.shape == object {
			b.Write(n.keys[i].raw)
			b.WriteString(": ")
			n.keys[i].val.write(b, indent, depth+1)
			continue
		}
		n.items[i].write(b, indent, depth+1)
	}
	newline(b, indent, depth)
	b.WriteByte(end)
}

// value: a line break, then the file's indent once per level: JSON's own
// nesting, which is no layout of Machines' output.
func newline(b *bytes.Buffer, l layout, depth int) {
	b.WriteString(l.eol)
	for range depth {
		b.WriteString(l.indent)
	}
}

// value: the value's meaning alone, for equality and digests: compact,
// keys sorted, a string by its decoded text, a number by its spelling.
func (n *node) canonical() []byte {
	var b bytes.Buffer
	n.canon(&b)
	return b.Bytes()
}

func (n *node) canon(b *bytes.Buffer) {
	switch n.shape {
	case scalar:
		var s string
		if n.raw[0] == '"' && json.Unmarshal(n.raw, &s) == nil {
			out, _ := json.Marshal(s)
			b.Write(out)
			return
		}
		b.Write(n.raw)
	case array:
		b.WriteByte('[')
		for i, it := range n.items {
			if i > 0 {
				b.WriteByte(',')
			}
			it.canon(b)
		}
		b.WriteByte(']')
	case object:
		keys := slices.Clone(n.keys)
		slices.SortFunc(keys, func(a, c member) int { return strings.Compare(a.name, c.name) })
		b.WriteByte('{')
		for i, m := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			name, _ := json.Marshal(m.name)
			b.Write(name)
			b.WriteByte(':')
			m.val.canon(b)
		}
		b.WriteByte('}')
	}
}

func (n *node) digest() string { return digestOf(n.canonical()) }

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (n *node) equal(o *node) bool { return bytes.Equal(n.canonical(), o.canonical()) }

// value: the indent of the file's first indented line, two spaces for a
// file with none, and its line end: `\r\n` when every line ends so, `\n`
// otherwise. A file that mixes both is refused: its line ends cannot be
// kept, and are never converted.
func layoutOf(to string, data []byte) (layout, error) {
	l := layout{indent: "  ", eol: "\n"}
	lines, crlf := bytes.Count(data, []byte("\n")), bytes.Count(data, []byte("\r\n"))
	switch {
	case crlf > 0 && crlf < lines:
		return layout{}, fmt.Errorf("%s mixes CRLF and LF line ends: Machines writes nothing into it", to)
	case crlf > 0:
		l.eol = "\r\n"
	}
	for _, line := range bytes.Split(data, []byte("\n"))[1:] {
		if ws := line[:len(line)-len(bytes.TrimLeft(line, " \t"))]; len(ws) > 0 {
			l.indent = string(ws)
			break
		}
	}
	return l, nil
}

// value: RFC 6901's pointer of a key under a parent's.
func child(parent, name string) string {
	return parent + "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}

// value: the node a pointer names, and its parent object with the key's
// index there; nil when the path is not there or crosses no object.
func (n *node) at(pointer string) (found, parent *node, index int) {
	if pointer == "" {
		return n, nil, -1
	}
	cur := n
	for _, part := range strings.Split(pointer[1:], "/") {
		name := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		if cur.shape != object {
			return nil, nil, -1
		}
		i := cur.member(name)
		if i < 0 {
			return nil, nil, -1
		}
		parent, index, cur = cur, i, cur.keys[i].val
	}
	return cur, parent, index
}
