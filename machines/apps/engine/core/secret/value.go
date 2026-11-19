package secret

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Value struct {
	// invariant: a secret station holds prints, formats and marshals as its
	// reference (`vault:<name>`), or Marker with none, never in clear; it
	// opens only through Reveal, and it is never read from JSON. The clear
	// bytes live in a closure: fmt prints a func as an address at any depth
	// and under any verb, so a Value in an unexported field, printed without
	// its methods, still shows no byte.
	ref  string
	open func() []byte
}

func Hold(ref string, plain []byte) Value {
	kept := append([]byte(nil), plain...)
	return Value{ref: ref, open: func() []byte { return kept }}
}

func (v Value) Reveal() []byte {
	// value: the only way to the clear bytes, a copy, for the point that
	// writes them.
	if v.open == nil {
		return nil
	}
	return append([]byte(nil), v.open()...)
}

func (v Value) String() string {
	if v.ref == "" {
		return Marker
	}
	return v.ref
}

func (v Value) GoString() string {
	return "secret.Value(" + strconv.Quote(v.String()) + ")"
}

func (v Value) Format(f fmt.State, verb rune) {
	s := v.String()
	switch {
	case verb == 'q':
		s = strconv.Quote(s)
	case verb == 'v' && f.Flag('#'):
		s = v.GoString()
	}
	_, _ = f.Write([]byte(s))
}

func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func (v Value) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v Value) MarshalYAML() (any, error) {
	return v.String(), nil
}
