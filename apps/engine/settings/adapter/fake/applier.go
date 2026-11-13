package fake

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Applier struct {
	w    *world.World
	mac  string
	kind catalog.EffectKind
}

var _ contract.Applier = (*Applier)(nil)

const Ours = "station's"

func ApplierOf(w *world.World, mac string, kind catalog.EffectKind) *Applier {
	return &Applier{w: w, mac: mac, kind: kind}
}

func NewApplier(kind catalog.EffectKind) *Applier {
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac")
	return ApplierOf(w, "this-mac", kind)
}

func (a *Applier) Kind() catalog.EffectKind { return a.kind }

func (a *Applier) held() map[string]string {
	m := a.w.Mac(a.mac)
	if m.Settings == nil {
		m.Settings = map[string]string{}
	}
	return m.Settings
}

func (a *Applier) Edit(name, text string) {
	// value: the owner writes their own text in place of a setting's.
	a.w.Lock()
	defer a.w.Unlock()
	a.held()[name] = text
}

// value: what station sets for a setting, the catalogue's values; one that
// declares none sets station's own text.
func wants(it catalog.Setting) []settings.KeyValue {
	if sets := settings.Sets(it); len(sets) > 0 {
		return sets
	}
	return []settings.KeyValue{{Key: it.Name, Value: Ours}}
}

func encode(values []settings.KeyValue) string {
	lines := make([]string, len(values))
	for i, kv := range values {
		lines[i] = kv.Key + "=" + kv.Value
	}
	return strings.Join(lines, "\n")
}

func decode(name, text string) []settings.KeyValue {
	var out []settings.KeyValue
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return []settings.KeyValue{{Key: name, Value: text}}
		}
		out = append(out, settings.KeyValue{Key: k, Value: v})
	}
	return out
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	a.w.Lock()
	defer a.w.Unlock()
	want := wants(it)
	text, ok := a.held()[it.Name]
	if !ok {
		return settings.Observation{State: catalog.StateAbsent, After: want}, nil
	}
	o := settings.Observation{State: catalog.StateOK, Digest: text, Before: decode(it.Name, text), After: want}
	if text != encode(want) {
		o.State, o.Detail = catalog.StateDegraded, "differs from station's"
	}
	return o, nil
}

func (a *Applier) Apply(_ context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	a.w.Lock()
	defer a.w.Unlock()
	a.held()[it.Name] = encode(wants(it))
	return settings.Change{Steps: []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: "applied"}}}, nil
}

func (a *Applier) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	a.w.Lock()
	defer a.w.Unlock()
	if a.held()[it.Name] != encode(wants(it)) {
		return settings.Change{Steps: []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: "yours, kept"}}}, nil
	}
	delete(a.held(), it.Name)
	return settings.Change{Steps: []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: "undone"}}}, nil
}
