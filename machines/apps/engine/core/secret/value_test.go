package secret_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

const plain = "hunter2-the-plain-text"

type carrier struct {
	Shown  secret.Value
	hidden secret.Value
}

func printed(v secret.Value) map[string]string {
	c, p := carrier{Shown: v, hidden: v}, &v
	out := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d", "%10s"} {
		out[verb] = fmt.Sprintf(verb, v)
		out[verb+" pointer"] = fmt.Sprintf(verb, p)
		out[verb+" in a struct"] = fmt.Sprintf(verb, c)
		out[verb+" in a slice"] = fmt.Sprintf(verb, []secret.Value{v})
		out[verb+" in a map"] = fmt.Sprintf(verb, map[string]secret.Value{"k": v})
	}
	out["Sprint"], out["Sprintln"] = fmt.Sprint(any(v)), fmt.Sprintln(v)
	out["an error"] = errors.New(fmt.Sprint(any(v))).Error()
	out["a wrapped error"] = fmt.Errorf("apply: %v", v).Error()
	for name, m := range map[string]any{"json": v, "json in a struct": c, "json in a map": map[string]secret.Value{"k": v}} {
		b, err := json.Marshal(m)
		if err != nil {
			out[name] = "error " + err.Error()
			continue
		}
		out[name] = string(b)
	}
	text, _ := v.MarshalText()
	out["text"] = string(text)
	if y, ok := any(v).(interface{ MarshalYAML() (any, error) }); ok {
		got, _ := y.MarshalYAML()
		out["yaml"] = fmt.Sprint(got)
	} else {
		out["yaml"] = "no MarshalYAML: " + plain
	}
	return out
}

func TestAValueNeverPrintsOrMarshalsInClear(t *testing.T) {
	for _, v := range []secret.Value{secret.Hold("vault:agent-password", []byte(plain)), secret.Hold("", []byte(plain))} {
		for how, s := range printed(v) {
			if strings.Contains(s, plain) || strings.Contains(s, fmt.Sprintf("%v", []byte(plain))) || strings.Contains(s, fmt.Sprintf("%x", plain)) {
				t.Errorf("%s printed the plain text: %q", how, s)
			}
		}
	}
	v := secret.Hold("vault:agent-password", []byte(plain))
	if got := fmt.Sprintf("%v %q %#v", v, v, v); got != `vault:agent-password "vault:agent-password" secret.Value("vault:agent-password")` {
		t.Errorf("a value does not print its reference: %s", got)
	}
	for _, verb := range []string{"%s", "%+v", "%d", "%x"} {
		if got := fmt.Sprintf(verb, v); got != "vault:agent-password" {
			t.Errorf("%s prints %q, not the reference", verb, got)
		}
	}
	if b, _ := json.Marshal(secret.Hold("", []byte(plain))); string(b) != `"withheld"` {
		t.Errorf("a value with no reference marshals as %s", b)
	}
}

func TestAValueOpensOnlyThroughReveal(t *testing.T) {
	v := secret.Hold("vault:x", []byte(plain))
	got := v.Reveal()
	if string(got) != plain {
		t.Fatalf("Reveal = %q", got)
	}
	got[0] = 'X'
	if !bytes.Equal(v.Reveal(), []byte(plain)) {
		t.Error("changing what Reveal returned changed the value")
	}
	if (secret.Value{}).Reveal() != nil {
		t.Error("an empty value revealed bytes")
	}
}

func TestAValueRidesNoInput(t *testing.T) {
	var in struct{ V secret.Value }
	if err := json.Unmarshal([]byte(`{"V":"`+plain+`"}`), &in); err == nil {
		t.Errorf("a value was read from JSON: %v", in.V.Reveal())
	}
}
