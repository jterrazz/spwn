package wiki

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// value: the deepest a braced value may nest its arrays and objects.
const maxDepth = 32

type lineIndex struct {
	src    string
	starts []int
}

func indexLines(src string) lineIndex {
	starts := []int{0}
	for i := range len(src) {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return lineIndex{src: src, starts: starts}
}

func (x lineIndex) at(off int) At {
	line := sort.Search(len(x.starts), func(i int) bool { return x.starts[i] > off }) - 1
	return At{Line: line + 1, Col: utf8.RuneCountInString(x.src[x.starts[line]:off]) + 1}
}

type tag struct {
	name  string
	props Props
	opens bool
	at    At
	end   int
}

type refusal struct {
	rule string
	at   At
	says string
}

type tagParser struct {
	x    lineIndex
	src  string
	pos  int
	name string
	at   At
}

// invariant: a tag is read, never run: a braced value is a literal, ref,
// query or state by the grammar of PLAN.md §3, and anything else in braces is
// refused (H39) with its text quoted. Nothing here evaluates what it reads.
func parseTag(x lineIndex, off int) (tag, *refusal) {
	p := &tagParser{x: x, src: x.src, pos: off + 1, at: x.at(off)}
	start := p.pos
	for p.pos < len(p.src) && isAlnum(p.src[p.pos]) {
		p.pos++
	}
	p.name = p.src[start:p.pos]
	t := tag{name: p.name, at: p.at}
	for {
		spaced, blank := p.skipSpace()
		switch {
		case strings.HasPrefix(p.src[p.pos:], "/>"):
			t.end = p.pos + 2
			return t, nil
		case strings.HasPrefix(p.src[p.pos:], ">"):
			t.opens, t.end = true, p.pos+1
			return t, nil
		case p.pos >= len(p.src) || blank && !isLower(p.src[p.pos]):
			return t, p.neverClosed()
		case !isLower(p.src[p.pos]):
			return t, p.refuse(p.pos, fmt.Sprintf("expected an attribute or the tag's end, found %q", nextRune(p.src[p.pos:])))
		case !spaced:
			return t, p.refuse(p.pos, "an attribute is set apart from what precedes it by a space")
		}
		prop, r := p.attribute()
		if r != nil {
			return t, r
		}
		if _, twice := t.props.Get(prop.Name); twice {
			return t, &refusal{rule: "H38", at: prop.At, says: fmt.Sprintf("<%s at line %d: %q is written twice", p.name, p.at.Line, prop.Name)}
		}
		t.props = append(t.props, prop)
	}
}

func (p *tagParser) closer() string {
	if p.name == "Group" {
		return ">"
	}
	return "/>"
}

func (p *tagParser) neverClosed() *refusal {
	return &refusal{rule: "H38", at: p.at, says: fmt.Sprintf("<%s at line %d is never closed: end it with %s", p.name, p.at.Line, p.closer())}
}

func (p *tagParser) refuse(off int, says string) *refusal {
	return &refusal{rule: "H38", at: p.x.at(off), says: fmt.Sprintf("<%s at line %d: %s", p.name, p.at.Line, says)}
}

// value: whether any space was skipped, and whether a blank line was among it.
func (p *tagParser) skipSpace() (spaced, blank bool) {
	newlines := 0
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\r':
		case '\n':
			newlines++
			if newlines > 1 {
				blank = true
			}
		default:
			return spaced, blank
		}
		spaced = true
		p.pos++
	}
	return spaced, blank
}

func (p *tagParser) attribute() (Prop, *refusal) {
	start := p.pos
	for p.pos < len(p.src) && isNameByte(p.src[p.pos]) {
		p.pos++
	}
	prop := Prop{Name: p.src[start:p.pos], At: p.x.at(start)}
	mark := p.pos
	p.skipSpace()
	if p.pos >= len(p.src) || p.src[p.pos] != '=' {
		p.pos = mark
		prop.Value = Value{Kind: ValueBool, Bool: true, At: prop.At}
		return prop, nil
	}
	p.pos++
	p.skipSpace()
	if p.pos >= len(p.src) {
		return prop, p.neverClosed()
	}
	switch p.src[p.pos] {
	case '"':
		v, r := p.attributeString(prop.Name)
		prop.Value = v
		return prop, r
	case '{':
		v, r := p.braced()
		prop.Value = v
		return prop, r
	}
	return prop, p.refuse(p.pos, fmt.Sprintf("the value of %q is a string \"...\" or a braced value {...}", prop.Name))
}

func (p *tagParser) attributeString(name string) (Value, *refusal) {
	open := p.pos
	text, ok := p.stringBody()
	if !ok {
		return Value{}, p.refuse(open, fmt.Sprintf("the string of %q is never terminated: a string ends with \" on the line it opens", name))
	}
	return Value{Kind: ValueString, Text: text, At: p.x.at(open)}, nil
}

// value: a string's characters between its quotes; no escape, no newline.
func (p *tagParser) stringBody() (string, bool) {
	start := p.pos + 1
	for i := start; i < len(p.src); i++ {
		switch p.src[i] {
		case '"':
			p.pos = i + 1
			return p.src[start:i], true
		case '\n':
			return "", false
		}
	}
	return "", false
}

func (p *tagParser) braced() (Value, *refusal) {
	open := p.pos
	end, ok := matchingBrace(p.src, open)
	if !ok {
		return Value{}, p.neverClosed()
	}
	expression := &refusal{rule: "H39", at: p.x.at(open), says: collapse(p.src[open:end+1]) + " is an expression: in braces, a value, ref(...) or query(...), nothing else"}
	e := &exprParser{x: p.x, src: p.src[:end], pos: open + 1}
	e.space()
	v, ok := e.expr(0, false)
	if !ok {
		return Value{}, expression
	}
	e.space()
	if e.pos != end {
		return Value{}, expression
	}
	p.pos = end + 1
	return v, nil
}

// value: the offset of the brace that closes the one at open, strings
// skipped; false when the page ends first.
func matchingBrace(src string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '"':
			end := strings.IndexAny(src[i+1:], "\"\n")
			if end < 0 {
				return 0, false
			}
			i += 1 + end
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

type exprParser struct {
	x   lineIndex
	src string
	pos int
}

func (e *exprParser) space() {
	for e.pos < len(e.src) && strings.IndexByte(" \t\r\n", e.src[e.pos]) >= 0 {
		e.pos++
	}
}

func (e *exprParser) eat(s string) bool {
	e.space()
	if strings.HasPrefix(e.src[e.pos:], s) {
		e.pos += len(s)
		return true
	}
	return false
}

func (e *exprParser) peek() byte {
	if e.pos < len(e.src) {
		return e.src[e.pos]
	}
	return 0
}

// invariant: `state(...)` is a value only as a condition's operand (H39
// elsewhere), so operand is true only there.
func (e *exprParser) expr(depth int, operand bool) (Value, bool) {
	if depth > maxDepth {
		return Value{}, false
	}
	e.space()
	at := e.x.at(e.pos)
	c := e.peek()
	switch {
	case c == '"':
		text, ok := e.str()
		return Value{Kind: ValueString, Text: text, At: at}, ok
	case c == '-' || isDigit(c):
		text, ok := e.number()
		return Value{Kind: ValueNumber, Text: text, At: at}, ok
	case c == '[':
		e.pos++
		list, ok := e.list(depth)
		return Value{Kind: ValueList, List: list, At: at}, ok
	case c == '{':
		e.pos++
		members, ok := e.object(depth)
		return Value{Kind: ValueObject, Members: members, At: at}, ok
	}
	word := e.word()
	switch word {
	case "true", "false":
		return Value{Kind: ValueBool, Bool: word == "true", At: at}, true
	case "null":
		return Value{Kind: ValueNull, At: at}, true
	case "ref":
		text, ok := e.call()
		return Value{Kind: ValueRef, Text: text, At: at}, ok
	case "state":
		text, ok := e.call()
		return Value{Kind: ValueState, Text: text, At: at}, ok && operand
	case "query":
		q, ok := e.query()
		return Value{Kind: ValueQuery, Query: q, At: at}, ok
	}
	return Value{}, false
}

func (e *exprParser) word() string {
	start := e.pos
	for e.pos < len(e.src) && isNameByte(e.src[e.pos]) {
		e.pos++
	}
	return e.src[start:e.pos]
}

func (e *exprParser) str() (string, bool) {
	start := e.pos + 1
	for i := start; i < len(e.src); i++ {
		switch e.src[i] {
		case '"':
			e.pos = i + 1
			return e.src[start:i], true
		case '\n':
			return "", false
		}
	}
	return "", false
}

func (e *exprParser) digits() bool {
	start := e.pos
	for e.pos < len(e.src) && isDigit(e.src[e.pos]) {
		e.pos++
	}
	return e.pos > start
}

func (e *exprParser) number() (string, bool) {
	start := e.pos
	if e.peek() == '-' {
		e.pos++
	}
	if !e.digits() {
		return "", false
	}
	if e.peek() == '.' {
		e.pos++
		if !e.digits() {
			return "", false
		}
	}
	return e.src[start:e.pos], true
}

func (e *exprParser) list(depth int) ([]Value, bool) {
	out := []Value{}
	if e.eat("]") {
		return out, true
	}
	for {
		v, ok := e.expr(depth+1, false)
		if !ok {
			return nil, false
		}
		out = append(out, v)
		if e.eat("]") {
			return out, true
		}
		if !e.eat(",") {
			return nil, false
		}
		if e.eat("]") {
			return out, true
		}
	}
}

func (e *exprParser) key() (string, bool) {
	e.space()
	if e.peek() == '"' {
		return e.str()
	}
	if !isLower(e.peek()) {
		return "", false
	}
	return e.word(), true
}

func (e *exprParser) object(depth int) ([]Member, bool) {
	out := []Member{}
	if e.eat("}") {
		return out, true
	}
	for {
		k, ok := e.key()
		if !ok || !e.eat(":") {
			return nil, false
		}
		v, ok := e.expr(depth+1, false)
		if !ok {
			return nil, false
		}
		out = append(out, Member{Key: k, Value: v})
		if e.eat("}") {
			return out, true
		}
		if !e.eat(",") {
			return nil, false
		}
		if e.eat("}") {
			return out, true
		}
	}
}

func (e *exprParser) call() (string, bool) {
	if !e.eat("(") {
		return "", false
	}
	e.space()
	if e.peek() != '"' {
		return "", false
	}
	text, ok := e.str()
	return text, ok && e.eat(")")
}

func (e *exprParser) query() (*Query, bool) {
	if !e.eat("(") || !e.eat("{") {
		return nil, false
	}
	e.space()
	if !isLower(e.peek()) {
		return nil, false
	}
	q := &Query{Field: e.word()}
	if !e.eat(":") || !e.eat("{") {
		return nil, false
	}
	e.space()
	switch op := Op(e.word()); op {
	case OpHas, OpIs:
		q.Op = op
	default:
		return nil, false
	}
	if !e.eat(":") {
		return nil, false
	}
	operand, ok := e.expr(1, true)
	if !ok || (operand.Kind != ValueString && operand.Kind != ValueRef && operand.Kind != ValueState) {
		return nil, false
	}
	q.Operand = operand
	return q, e.closes("}", "}", ")")
}

func nextRune(s string) string {
	r, _ := utf8.DecodeRuneInString(s)
	return string(r)
}

func isLower(c byte) bool    { return c >= 'a' && c <= 'z' }
func isUpper(c byte) bool    { return c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool    { return c >= '0' && c <= '9' }
func isAlnum(c byte) bool    { return isLower(c) || isUpper(c) || isDigit(c) }
func isNameByte(c byte) bool { return isLower(c) || isDigit(c) || c == '_' }

func (e *exprParser) closes(marks ...string) bool {
	for _, m := range marks {
		if !e.eat(m) {
			return false
		}
	}
	return true
}
