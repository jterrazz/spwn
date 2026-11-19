package secret

import "encoding/json"

type Sent struct {
	// invariant: a secret the owner typed into a request: it reads from the
	// request's JSON once, and from then on prints, formats and marshals as
	// its marker, so no answer, log or error that holds it shows a byte.
	Value
}

func (Sent) WireSchema() map[string]any {
	return map[string]any{"type": "string", "writeOnly": true}
}

func (s *Sent) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return err
	}
	*s = Sent{Hold("", []byte(text))}
	return nil
}
