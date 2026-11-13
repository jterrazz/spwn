package secret

import (
	"bytes"
	"encoding/json"
	"slices"
	"sync/atomic"
)

type armed struct {
	secrets [][]byte
	trip    func(said string)
}

// invariant: the tripwire is armed by the world build alone (wire's
// world_on.go), whose trip panics; the release build never arms it, so
// Tripwire does nothing there.
var tripwire atomic.Pointer[armed]

func Arm(secrets []string, trip func(said string)) {
	a := &armed{trip: trip}
	for _, s := range secrets {
		if s != "" {
			a.secrets = append(a.secrets, encodings(s)...)
		}
	}
	tripwire.Store(a)
}

func encodings(s string) [][]byte {
	// invariant: a secret is looked for as written and as each JSON encoder
	// writes it inside a string, HTML escaped or not, so an answer encoded
	// after the secret entered it still trips.
	forms := [][]byte{[]byte(s)}
	for _, html := range []bool{true, false} {
		var b bytes.Buffer
		e := json.NewEncoder(&b)
		e.SetEscapeHTML(html)
		if e.Encode(s) != nil {
			continue
		}
		form := bytes.TrimSuffix(bytes.TrimSpace(b.Bytes()), []byte(`"`))
		form = bytes.TrimPrefix(form, []byte(`"`))
		if !slices.ContainsFunc(forms, func(f []byte) bool { return bytes.Equal(f, form) }) {
			forms = append(forms, form)
		}
	}
	return forms
}

func Disarm() { tripwire.Store(nil) }

func Tripwire(answer []byte) {
	// invariant: once armed, an answer that holds one of the world's secrets
	// or a shape that names its issuer trips, said with its secrets
	// withheld: every story is a leak check.
	a := tripwire.Load()
	if a == nil {
		return
	}
	leaked := issued.Match(answer)
	for _, s := range a.secrets {
		if bytes.Contains(answer, s) {
			leaked = true
			answer = bytes.ReplaceAll(answer, s, []byte(Marker))
		}
	}
	if !leaked {
		return
	}
	said, _ := Screen(string(answer))
	if len(said) > 600 {
		said = said[:600] + "..."
	}
	a.trip("an answer carries a secret: " + said)
}

func TripwireJSON(answer any) {
	// value: Tripwire over a value's JSON, marshaled only once armed.
	if tripwire.Load() == nil {
		return
	}
	data, err := json.Marshal(answer)
	if err != nil {
		return
	}
	Tripwire(data)
}
