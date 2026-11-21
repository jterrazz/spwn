package store

import (
	"encoding/json"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

type View struct {
	// value: a projection of one snapshot a client reads by name beside
	// the slices: its value, when the facts it is made of were read, and
	// whether they are fresh.
	Value  any
	ReadAt time.Time
	State  State
	Why    string
}

func (s *Store) Encode(v any) (raw json.RawMessage, hash string, err error) {
	// invariant: a view is encoded as a slice is, its secrets withheld
	// before any client holds its bytes, and hashed the same way.
	raw, err = call.Encode(v, s.word)
	if err != nil {
		return nil, "", err
	}
	return raw, hashOf(raw), nil
}
