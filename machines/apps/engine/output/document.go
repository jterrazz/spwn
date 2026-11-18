package output

import (
	"bytes"
	"encoding/json"
	"time"
)

type Machine struct {
	// value: the machine a document is about, as every document names it
	// (SPEC §11.1); On is a VM's Mac, null for a Mac.
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Kind    string  `json:"kind"`
	On      *string `json:"on"`
	ThisMac bool    `json:"this_mac"`
}

type Envelope struct {
	At      time.Time
	Machine Machine
}

// value: a document kept for one release in the shape it had (SPEC §11.4):
// the connector that reads it decodes its fields, not the envelope.
func kept(schema string) bool { return schema == "station.status/v1" }

type member struct {
	key   string
	value json.RawMessage
}

func Document(contract any, e *Envelope) ([]byte, error) {
	// invariant: a document opens with schema, generated_at, machine and
	// findings, in that order, then its call's body in the order it gave it;
	// what the call already said of them is kept, the envelope only fills in.
	body, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	members, ok := object(body)
	if !ok || e == nil {
		return indent(body)
	}
	got := map[string]json.RawMessage{}
	for _, m := range members {
		got[m.key] = m.value
	}
	var schema string
	if json.Unmarshal(got["schema"], &schema) != nil || schema == "" || schema == ErrorSchema || kept(schema) {
		return indent(body)
	}
	at, _ := json.Marshal(e.At.UTC().Format(time.RFC3339))
	head := []member{{"schema", got["schema"]}, {"generated_at", firstOf(got["generated_at"], at)}, {"machine", machineOf(got["machine"], e.Machine)}, {"findings", firstOf(got["findings"], json.RawMessage("[]"))}}
	out := head
	for _, m := range members {
		switch m.key {
		case "schema", "generated_at", "machine", "findings":
			continue
		}
		out = append(out, m)
	}
	return indent(join(out))
}

func firstOf(have, otherwise json.RawMessage) json.RawMessage {
	if len(have) == 0 || string(have) == "null" {
		return otherwise
	}
	return have
}

func machineOf(have json.RawMessage, m Machine) json.RawMessage {
	// invariant: the call's own words about its machine win (a VM's Mac, its
	// kind); the envelope adds the keys it left out.
	base, _ := json.Marshal(m)
	members, ok := object(have)
	if !ok {
		return base
	}
	merged, _ := object(base)
	for _, own := range members {
		found := false
		for i := range merged {
			if merged[i].key == own.key {
				merged[i].value, found = own.value, true
			}
		}
		if !found {
			merged = append(merged, own)
		}
	}
	return join(merged)
}

func join(members []member) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		k, _ := json.Marshal(m.key)
		b.Write(k)
		b.WriteByte(':')
		b.Write(m.value)
	}
	b.WriteByte('}')
	return b.Bytes()
}

func object(data []byte) ([]member, bool) {
	d := json.NewDecoder(bytes.NewReader(data))
	if t, err := d.Token(); err != nil || t != json.Delim('{') {
		return nil, false
	}
	var out []member
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return nil, false
		}
		key, _ := t.(string)
		var v json.RawMessage
		if err := d.Decode(&v); err != nil {
			return nil, false
		}
		out = append(out, member{key, v})
	}
	return out, true
}

func indent(data []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := json.Indent(&b, data, "", "  "); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}
