package wiki

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// value: whether a character draws nothing or reorders the text around it:
// a C0 control but the line feed and the tab, DEL, a C1 control, a bidi
// embedding, override or isolate, and the line and paragraph separators.
// The joiners and variation selectors an emoji is drawn with are kept.
func silent(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return r == 0x2028 || r == 0x2029
}

// invariant: text enters the compiled model only through audible: a page's
// id and source, which no step decodes, and every string wiki.yaml's
// decoder produced (audibleAll) leave it with no silent character, and say
// how many it removed. A text that holds none, as every honest one does,
// is scanned once and returned as it is.
func audible(text string) (kept string, removed int) {
	if !strings.ContainsFunc(text, silent) {
		return text, 0
	}
	n := 0
	return strings.Map(func(r rune) rune {
		if silent(r) {
			n++
			return -1
		}
		return r
	}, text), n
}

func Audible(text string) string {
	// value: text as an answer may carry it, its silent characters removed: a
	// path the folder could not read is named through it, like a page's id.
	kept, _ := audible(text)
	return kept
}

// invariant: a decoded value leaves audibleAll with every string it holds,
// at any depth and as a map key too, passed through audible, so an escape
// its decoder spelled is removed like a raw character; it says how many it
// removed. Map keys are visited in their sorted order.
func audibleAll(v reflect.Value) (removed int) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return audibleAll(v.Elem())
		}
	case reflect.String:
		kept, n := audible(v.String())
		if n > 0 && v.CanSet() {
			v.SetString(kept)
		}
		return n
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				removed += audibleAll(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			removed += audibleAll(v.Index(i))
		}
	case reflect.Map:
		keys := v.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		for _, k := range keys {
			key := reflect.New(k.Type()).Elem()
			key.Set(k)
			value := reflect.New(v.Type().Elem()).Elem()
			value.Set(v.MapIndex(k))
			n := audibleAll(key) + audibleAll(value)
			if n > 0 {
				v.SetMapIndex(k, reflect.Value{})
				v.SetMapIndex(key, value)
			}
			removed += n
		}
	default:
	}
	return removed
}

// value: the finding that says a file held characters the engine removed.
func silenced(page string, n int) Finding {
	return finding("S32", page, "", At{Line: 1, Col: 1}, fmt.Sprintf("%d characters that draw nothing or reorder text were removed", n))
}
