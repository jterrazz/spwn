package api

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
)

const (
	// value: the first release whose station writes computer, ephemeral and
	// persistent (ADR-032); a peer older than it reads only the words before.
	WordsSince = "v3.1.0"
)

func Older(version, own string) bool {
	// value: whether a peer that announced version reads only the words
	// before WordsSince.
	// invariant: for one release, a peer is older when its Hello names no
	// version (the hello an older station sends) or a release before
	// WordsSince; a peer that runs this very build, or one whose version
	// names no release (a dev build), reads today's words. A build between
	// two releases counts as the release it follows: the old words it is
	// sent are words it still reads.
	if version == own && own != "" {
		return false
	}
	if version == "" {
		return true
	}
	at, ok := release(version)
	if !ok {
		return false
	}
	since, _ := release(WordsSince)
	for i := range at {
		if at[i] != since[i] {
			return at[i] < since[i]
		}
	}
	return false
}

// value: the major, minor and patch of a vX.Y.Z version, whatever follows
// its patch; ok is false for a version that names no release.
func release(version string) (at [3]int, ok bool) {
	core, found := strings.CutPrefix(version, "v")
	if !found {
		return at, false
	}
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return at, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return at, false
		}
		at[i] = n
	}
	return at, true
}

// value: the word an older station reads in place of today's word, for the
// field that carries it; ok is false for any other field or word.
func olderWord(field, word string) (then string, ok bool) {
	switch {
	case field == "kind" && word == string(machine.Computer):
		return machine.OlderComputer, true
	case field == "lifetime" && word == string(machines.Ephemeral):
		return machines.OlderEphemeral, true
	case field == "lifetime" && word == string(machines.Persistent):
		return machines.OlderPersistent, true
	}
	return "", false
}

func OlderWords(raw json.RawMessage) json.RawMessage {
	// value: raw with every kind and lifetime written in the words an older
	// station reads.
	// invariant: only the value of a field named kind or lifetime (in any
	// case) that holds one of today's words changes; every other byte keeps
	// its meaning, numbers their digits. A value that is not JSON is
	// returned as it came.
	if !mentions(raw) {
		return raw
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return raw
	}
	out, err := json.Marshal(older(v))
	if err != nil {
		return raw
	}
	return out
}

func mentions(raw json.RawMessage) bool {
	for _, now := range []machine.Kind{machine.Computer} {
		if bytes.Contains(raw, []byte(`"`+now+`"`)) {
			return true
		}
	}
	for _, now := range []machines.Lifetime{machines.Ephemeral, machines.Persistent} {
		if bytes.Contains(raw, []byte(`"`+now+`"`)) {
			return true
		}
	}
	return false
}

func older(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if word, ok := value.(string); ok {
				if then, ok := olderWord(strings.ToLower(key), word); ok {
					v[key] = then
					continue
				}
			}
			v[key] = older(value)
		}
	case []any:
		for i := range v {
			v[i] = older(v[i])
		}
	}
	return v
}
