package call

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

type WireError struct {
	Type    fail.Type         `json:"type,omitempty"`
	Code    fail.Code         `json:"code,omitempty"`
	Subject string            `json:"subject,omitempty"`
	Detail  map[string]string `json:"detail,omitempty"`
	Message string            `json:"message"`
	Hint    string            `json:"hint,omitempty"`
	Exit    fail.Exit         `json:"exit,omitempty"`
	Fix     *finding.Fix      `json:"fix,omitempty"`
}

type Wording func(error) *fail.Error

func ToWire(err error, word Wording) *WireError {
	// invariant: the far side words its error once, with its own table; the
	// code travels beside the words, so a caller that knows the code words it
	// again with its own, and one that does not prints these.
	if err == nil {
		return nil
	}
	f := word(err)
	if f == nil {
		f = &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
	}
	w := &WireError{Type: f.Type, Code: f.Code, Subject: screened(f.Subject), Detail: screenedAll(f.Detail), Message: screened(f.Message), Hint: screened(f.Hint), Exit: f.Exit}
	if w.Message == "" {
		w.Message = screened(f.Error())
	}
	if f.Fix.Call != "" || f.Fix.Command != "" || f.Fix.Path != "" || len(f.Fix.Args) > 0 {
		fix := f.Fix
		w.Fix = &fix
	}
	return w
}

func screened(text string) string {
	kept, _ := secret.Screen(text)
	return kept
}

func screenedAll(detail map[string]string) map[string]string {
	if detail == nil {
		return nil
	}
	out := make(map[string]string, len(detail))
	for k, v := range detail {
		out[k] = screened(v)
	}
	return out
}

func (w *WireError) Err() *fail.Error {
	if w == nil {
		return nil
	}
	e := &fail.Error{Type: w.Type, Code: w.Code, Subject: w.Subject, Detail: w.Detail, Exit: w.Exit}
	if w.Fix != nil {
		e.Fix = *w.Fix
	}
	if w.Code == "" {
		e.Message, e.Hint = w.Message, w.Hint
		return e
	}
	e.Cause = &fail.Said{Message: w.Message, Hint: w.Hint}
	return e
}

func Encode(v any, word Wording) (json.RawMessage, error) {
	// invariant: the wire form of a call's value: JSON, where an error inside
	// it travels as its WireError, never as an empty object.
	if raw, ok := v.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(tree(reflect.ValueOf(v), word))
}

var (
	errorType     = reflect.TypeFor[error]()
	marshalerType = reflect.TypeFor[json.Marshaler]()
	textType      = reflect.TypeFor[encoding.TextMarshaler]()
)

func tree(v reflect.Value, word Wording) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		if err, ok := v.Interface().(error); ok && v.Type() == errorType {
			return ToWire(err, word)
		}
		return tree(v.Elem(), word)
	}
	if !holdsError(v.Type()) {
		return v.Interface()
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return tree(v.Elem(), word)
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		out := make([]any, v.Len())
		for i := range out {
			out[i] = tree(v.Index(i), word)
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := make(map[string]any, v.Len())
		for it := v.MapRange(); it.Next(); {
			out[keyText(it.Key())] = tree(it.Value(), word)
		}
		return out
	case reflect.Struct:
		out := map[string]any{}
		for _, f := range fields(v.Type()) {
			fv := v.FieldByIndex(f.index)
			if f.omitEmpty && fv.IsZero() {
				continue
			}
			out[f.name] = tree(fv, word)
		}
		return out
	default:
		return v.Interface()
	}
}

func keyText(k reflect.Value) string {
	if k.Kind() == reflect.String {
		return k.String()
	}
	b, _ := json.Marshal(k.Interface())
	return strings.Trim(string(b), `"`)
}

type field struct {
	name      string
	index     []int
	omitEmpty bool
}

func fields(t reflect.Type) []field {
	var out []field
	for _, f := range reflect.VisibleFields(t) {
		if !f.IsExported() || f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("json") == "" {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" && opts == "" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		out = append(out, field{name: name, index: f.Index, omitEmpty: strings.Contains(opts, "omitempty")})
	}
	return out
}

type EncodedKey struct {
	Name      string
	Type      reflect.Type
	OmitEmpty bool
}

func EncodedKeys(t reflect.Type) ([]EncodedKey, bool) {
	// value: the keys Encode writes for a struct it walks itself, one that
	// holds an error, and whether it does: a key is left out only when its
	// tag says omitempty and its value is zero, omitzero and `,string` are
	// not read. A struct Encode does not walk is encoding/json's.
	if t.Kind() != reflect.Struct || !holdsError(t) {
		return nil, false
	}
	var out []EncodedKey
	for _, f := range fields(t) {
		out = append(out, EncodedKey{Name: f.name, Type: t.FieldByIndex(f.index).Type, OmitEmpty: f.omitEmpty})
	}
	return out, true
}

func holdsError(t reflect.Type) bool { return reaches(t, map[reflect.Type]bool{}) }

func reaches(t reflect.Type, seen map[reflect.Type]bool) bool {
	if t == errorType {
		return true
	}
	if seen[t] || t.Implements(marshalerType) || t.Implements(textType) ||
		reflect.PointerTo(t).Implements(marshalerType) || reflect.PointerTo(t).Implements(textType) {
		return false
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return reaches(t.Elem(), seen)
	case reflect.Struct:
		for _, f := range fields(t) {
			if reaches(t.FieldByIndex(f.index).Type, seen) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func Decode(raw json.RawMessage, out any) error {
	// invariant: the inverse of Encode: a WireError lands in an error field
	// as the far side's *fail.Error.
	return decode(raw, reflect.ValueOf(out).Elem())
}

func decode(raw json.RawMessage, v reflect.Value) error {
	if v.Type() == errorType {
		if isNull(raw) {
			v.SetZero()
			return nil
		}
		var w WireError
		if err := json.Unmarshal(raw, &w); err != nil {
			return err
		}
		v.Set(reflect.ValueOf(error(w.Err())))
		return nil
	}
	if !holdsError(v.Type()) {
		return json.Unmarshal(raw, v.Addr().Interface())
	}
	if isNull(raw) {
		v.SetZero()
		return nil
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decode(raw, v.Elem())
	case reflect.Slice:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		s := reflect.MakeSlice(v.Type(), len(items), len(items))
		for i, item := range items {
			if err := decode(item, s.Index(i)); err != nil {
				return err
			}
		}
		v.Set(s)
	case reflect.Array:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for i := 0; i < v.Len() && i < len(items); i++ {
			if err := decode(items[i], v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		var items map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		m := reflect.MakeMapWithSize(v.Type(), len(items))
		for k, item := range items {
			key := reflect.New(v.Type().Key()).Elem()
			if err := json.Unmarshal([]byte(`"`+k+`"`), key.Addr().Interface()); err != nil {
				return err
			}
			val := reflect.New(v.Type().Elem()).Elem()
			if err := decode(item, val); err != nil {
				return err
			}
			m.SetMapIndex(key, val)
		}
		v.Set(m)
	case reflect.Struct:
		var items map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for _, f := range fields(v.Type()) {
			item, ok := items[f.name]
			if !ok {
				continue
			}
			if err := decode(item, v.FieldByIndex(f.index)); err != nil {
				return err
			}
		}
	default:
		return json.Unmarshal(raw, v.Addr().Interface())
	}
	return nil
}

func isNull(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null"
}
