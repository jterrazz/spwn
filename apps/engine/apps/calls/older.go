package calls

import (
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func OlderIn(name string, in json.RawMessage) (out json.RawMessage, lacks string) {
	// outside: a v3.0 station reads apps.update's managers in its own
	// words (`skill` is `skills`, `cask` is Homebrew's `brew`) and reads a
	// manager it has none of as no manager, so it would update nothing:
	// that one is lacks, the In left as it came. Every other call's In
	// is its own. Kept until the hidden --method goes.
	if name != Update.Name {
		return in, ""
	}
	var fields map[string]json.RawMessage
	var keys []string
	if json.Unmarshal(in, &fields) != nil || json.Unmarshal(fields["Method"], &keys) != nil || len(keys) == 0 {
		return in, ""
	}
	for i, key := range keys {
		word, ok := catalog.OlderManager(key)
		if !ok {
			return in, key
		}
		keys[i] = word
	}
	fields["Method"], _ = json.Marshal(keys)
	said, err := json.Marshal(fields)
	if err != nil {
		return in, ""
	}
	return said, ""
}
