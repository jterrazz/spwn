package wiki

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// value: the cap of a sentence whose field names none, SPEC-v1 §5.
const sentenceCap = 200

var (
	anchorForm  = regexp.MustCompile(`^([a-z][a-z0-9-]*):([^\s#:]+)(#[A-Za-z0-9_.-]+)?$`)
	dateForm    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	intForm     = regexp.MustCompile(`^-?\d+$`)
	blockIDForm = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// value: what is wrong with a value for a field's type, none when it fits;
// a union fits when one of its members does, a list when each element does.
func fit(t string, f Field, v Value) message {
	if inner, ok := listOf(t); ok {
		if v.Kind != ValueList {
			return sayf("%s is %s, it takes %s", f.Name, describe(v), t)
		}
		for _, e := range v.List {
			if problem := fit(inner, f, e); !problem.none() {
				return problem
			}
		}
		return message{}
	}
	var why message
	for _, base := range strings.Split(t, "|") {
		ok, problem := fitBase(base, f, v)
		if ok {
			return message{}
		}
		if why.none() {
			why = problem
		}
	}
	if !why.none() {
		return why
	}
	return sayf("%s is %s, it takes %s", f.Name, describe(v), t)
}

// value: whether a value fits one base type, and when it is of the base's
// shape but out of its range, what is wrong with it.
func fitBase(base string, f Field, v Value) (fits bool, problem message) {
	str := v.Kind == ValueString
	switch base {
	case "sentence":
		if !str {
			return false, message{}
		}
		limit := f.Max
		if limit == 0 {
			limit = sentenceCap
		}
		if n := utf8.RuneCountInString(v.Text); n > limit {
			return false, sayf("%s is %d characters, %d at most", f.Name, n, limit)
		}
		return true, message{}
	case "text", "glob":
		return str, message{}
	case "enum":
		if str && slices.Contains(f.Values, v.Text) {
			return true, message{}
		}
		return false, sayf("%s: %s does not exist, it is %s", f.Name, text(v), oneOf(f.Values))
	case "ref":
		return v.Kind == ValueRef, message{}
	case "anchor":
		return str && anchorForm.MatchString(v.Text), message{}
	case "date":
		return str && dateForm.MatchString(v.Text), message{}
	case "bool":
		return v.Kind == ValueBool, message{}
	case "int":
		return v.Kind == ValueNumber && intForm.MatchString(v.Text), message{}
	case "object":
		return v.Kind == ValueObject, message{}
	case "query":
		return v.Kind == ValueQuery, message{}
	default:
		return false, message{}
	}
}

func describe(v Value) string {
	switch v.Kind {
	case ValueString:
		return fmt.Sprintf("%q", v.Text)
	case ValueNumber:
		return "the number " + v.Text
	case ValueBool:
		return "a bool"
	case ValueNull:
		return "null"
	case ValueList:
		return "a list"
	case ValueObject:
		return "an object"
	case ValueRef:
		return "a ref"
	case ValueQuery:
		return "a query"
	case ValueState:
		return "a state"
	default:
		return "nothing"
	}
}

// value: `a, b or c`.
func orList(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " or " + words[len(words)-1]
}

// value: every ref a value holds, at any depth, a query's operand included.
func refsIn(v Value) []Value {
	switch v.Kind {
	case ValueRef:
		return []Value{v}
	case ValueList:
		var out []Value
		for _, e := range v.List {
			out = append(out, refsIn(e)...)
		}
		return out
	case ValueObject:
		var out []Value
		for _, m := range v.Members {
			out = append(out, refsIn(m.Value)...)
		}
		return out
	case ValueQuery:
		return refsIn(v.Query.Operand)
	default:
		return nil
	}
}

// value: the strings a field's value holds, itself or as a list's elements.
func stringsIn(v Value) []Value {
	switch v.Kind {
	case ValueString:
		return []Value{v}
	case ValueList:
		var out []Value
		for _, e := range v.List {
			out = append(out, stringsIn(e)...)
		}
		return out
	default:
		return nil
	}
}

// value: where a value was written, its attribute's own when the value
// carries none.
func whereOf(v Value, fallback At) At {
	if v.At.Line > 0 {
		return v.At
	}
	return fallback
}

func (p Props) at(name string, fallback At) At {
	for _, prop := range p {
		if prop.Name == name {
			return prop.At
		}
	}
	return fallback
}
