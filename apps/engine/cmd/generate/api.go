package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const (
	apiDir      = "packages/api"
	apiCallsDir = "packages/api/schemas/calls"
	framePath   = "packages/api/schemas/frame.json"
	eventPath   = "packages/api/schemas/event.json"
	typesPath   = "packages/api/index.d.ts"
	module      = "github.com/jterrazz/studio-station"
	draft       = "https://json-schema.org/draft/2020-12/schema"
)

// invariant: the shapes of remote/api's Target and Progress and of
// remote/contract's Hello, which cmd/generate may not import: remote/api's
// own test holds frame.json to the real types key by key.
type (
	frameTarget struct {
		Name string `json:"name,omitempty"`
		On   string `json:"on,omitempty"`
	}
	frameHello struct {
		Version string       `json:"version"`
		Calls   []string     `json:"calls"`
		Kind    machine.Kind `json:"kind,omitempty"`
	}
	frameProgress struct {
		Task   string          `json:"task"`
		Detail string          `json:"detail,omitempty"`
		Done   bool            `json:"done,omitempty"`
		Error  *call.WireError `json:"error,omitempty"`
	}
)

// invariant: the forms two types write with their own MarshalJSON, mirrored
// from catalog's wireLivesOn and activity's seriesJSON, which no other
// package sees: Install is the install of its Method, kept untyped.
type (
	livesOnJSON struct {
		Kinds   []catalog.MachineKind
		Method  catalog.Method  `json:",omitempty"`
		Install json.RawMessage `json:",omitempty"`
	}
	seriesJSON struct {
		Fine   []activity.Sample `json:"fine"`
		Coarse []activity.Sample `json:"coarse"`
	}
)

func mirrors() map[reflect.Type]reflect.Type {
	return map[reflect.Type]reflect.Type{
		reflect.TypeFor[catalog.LivesOn](): reflect.TypeFor[livesOnJSON](),
		reflect.TypeFor[activity.Series](): reflect.TypeFor[seriesJSON](),
	}
}

type eventShape struct {
	// value: the event's name in the schemas and the TypeScript.
	name string
	// value: the keys the event carries beside `event`, the required ones
	// first.
	required, optional []string
}

// value: the events a watch streams, by their `event`, in the order
// event.json lists them.
func events() []string { return []string{"page", engine.EventSample, "tick", "notice"} }

func eventOf(event string) eventShape {
	return map[string]eventShape{
		"page":             {"PageEvent", []string{"page"}, nil},
		engine.EventSample: {"SampleEvent", []string{"at"}, []string{"values", "stale"}},
		"tick":             {"TickEvent", []string{"at"}, nil},
		"notice":           {"NoticeEvent", []string{"notice"}, nil},
	}[event]
}

// value: every call of the wire: the registry's, then the engine's own, by name.
type wireCall struct {
	meta   call.Meta
	opens  reflect.Type
	events []string
	own    bool
}

func wireCalls() []wireCall {
	owns := enginecalls.Owns()
	metas := registry().Metas()
	out := make([]wireCall, 0, len(metas)+len(owns))
	for _, m := range metas {
		out = append(out, wireCall{meta: m})
	}
	for _, o := range owns {
		out = append(out, wireCall{meta: o.Meta, opens: o.Opens, events: o.Events, own: true})
	}
	slices.SortFunc(out, func(a, b wireCall) int { return strings.Compare(a.meta.Name, b.meta.Name) })
	return out
}

type wireRules struct {
	custom map[reflect.Type]func(*reflector) map[string]any
	fields map[string]func(*reflector) map[string]any
	enums  map[string][]string
	// value: the struct types some answer carries, whose reading as an
	// input is a type of its own.
	answered map[reflect.Type]bool
}

func (w *wireRules) key(t reflect.Type, input bool) string {
	k := qualified(t)
	if input && w.answered[t] {
		return k + ".input"
	}
	return k
}

// value: a Go type's name in the schemas: its package under apps/engine/ and its
// name, `core.machine.Kind`.
func qualified(t reflect.Type) string {
	pkg := strings.TrimPrefix(t.PkgPath(), module+"/apps/engine/")
	if pkg == t.PkgPath() {
		panic("the wire schema names no type outside the module: " + t.String())
	}
	name := strings.ReplaceAll(t.Name(), module+"/apps/engine/", "")
	name = strings.Join(strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), ".")
	return strings.ReplaceAll(pkg, "/", ".") + "." + name
}

func newWireRules(root string) *wireRules {
	w := &wireRules{
		custom: map[reflect.Type]func(*reflector) map[string]any{
			reflect.TypeFor[time.Time](): func(*reflector) map[string]any {
				return map[string]any{"type": "string", "format": "date-time"}
			},
			reflect.TypeFor[json.RawMessage](): func(*reflector) map[string]any { return map[string]any{} },
			reflect.TypeFor[time.Location]():   func(*reflector) map[string]any { return closed(map[string]any{}, nil) },
		},
		fields: map[string]func(*reflector) map[string]any{
			"engine.Event.page": func(r *reflector) map[string]any { return r.wireOf(reflect.TypeFor[page.Document]()) },
			"engine.Event.values": func(r *reflector) map[string]any {
				return map[string]any{"type": "object", "additionalProperties": r.wireOf(reflect.TypeFor[page.Value]())}
			},
		},
		enums:    enumsOf(root),
		answered: map[reflect.Type]bool{},
	}
	for t, form := range mirrors() {
		w.custom[t] = func(r *reflector) map[string]any { return r.mirrored(t, form) }
	}
	return w
}

func (r *reflector) mirrored(t, form reflect.Type) map[string]any {
	key := r.wire.key(t, r.input)
	if _, seen := r.defs[key]; !seen {
		r.defs[key] = true
		r.defs[key] = r.wireObject(form)
	}
	return map[string]any{"$ref": "#/$defs/" + key}
}

func (w *wireRules) learnAnswers(calls []wireCall) {
	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		if seen[t] {
			return
		}
		if form, ok := mirrors()[t]; ok {
			seen[t], w.answered[t] = true, true
			walk(form)
			return
		}
		if w.custom[t] != nil {
			return
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			walk(t.Elem())
		case reflect.Interface:
			if t == errorType {
				walk(reflect.TypeFor[call.WireError]())
			}
		case reflect.Struct:
			if t.Name() != "" {
				w.answered[t] = true
			}
			for _, f := range encodedFields(t, false) {
				walk(f.t)
			}
		default:
		}
	}
	for _, c := range calls {
		walk(c.meta.OutType)
		if c.opens != nil {
			walk(c.opens)
		}
	}
	walk(reflect.TypeFor[page.Document]())
	walk(reflect.TypeFor[page.Value]())
	walk(reflect.TypeFor[engine.Notice]())
	walk(reflect.TypeFor[frameHello]())
	walk(reflect.TypeFor[frameProgress]())
}

func (r *reflector) event(name string) map[string]any {
	keys := eventOf(name)
	key := keys.name
	if _, seen := r.defs[key]; !seen {
		props := map[string]any{"event": map[string]any{"const": name}}
		fields := map[string]wireField{}
		for _, f := range wireFields(reflect.TypeFor[engine.Event]()) {
			fields[f.name] = f
		}
		for _, k := range slices.Concat(keys.required, keys.optional) {
			if over, ok := r.wire.fields["engine.Event."+k]; ok {
				props[k] = over(r)
				continue
			}
			props[k] = r.wireOf(fields[k].t)
		}
		r.defs[key] = closed(props, slices.Concat([]string{"event"}, keys.required))
	}
	return map[string]any{"$ref": "#/$defs/" + key}
}

func (r *reflector) out(c wireCall) map[string]any {
	r.input = false
	if c.opens == nil {
		return r.nullable(c.meta.OutType, r.zeroable(c.meta.OutType, r.wireOf(c.meta.OutType)))
	}
	first := []any{r.wireOf(c.opens)}
	for _, e := range c.events {
		first = append(first, r.event(e))
	}
	return map[string]any{"anyOf": first}
}

func (r *reflector) in(c wireCall) map[string]any {
	r.input = true
	defer func() { r.input = false }()
	return r.wireOf(c.meta.InType)
}

func callSchema(w *wireRules, c wireCall) map[string]any {
	r := &reflector{defs: map[string]any{}, wire: w}
	in := r.in(c)
	out := r.out(c)
	kind := "call"
	if c.meta.Watch {
		kind = "watch"
	}
	return map[string]any{
		"$schema":     draft,
		"$id":         "https://github.com/jterrazz/studio-station/blob/main/" + apiCallsDir + "/" + c.meta.Name + ".json",
		"title":       c.meta.Name,
		"description": "The " + kind + " " + c.meta.Name + " of station.api/v2: the `in` its request frame carries, and the `out` of each answer. Written by make generate from the Go types the call reads and answers; the frames are frame.json.",
		"type":        "object", "additionalProperties": false, "required": []string{"in", "out"},
		"properties": map[string]any{"in": in, "out": out},
		"$defs":      r.defs,
	}
}

func frameSchema(w *wireRules) map[string]any {
	r := &reflector{defs: map[string]any{}, wire: w}
	r.defs["Target"] = r.wireObject(reflect.TypeFor[frameTarget]())
	r.defs["Hello"] = r.wireObject(reflect.TypeFor[frameHello]())
	r.defs["Progress"] = r.wireObject(reflect.TypeFor[frameProgress]())
	wireError := r.wireOf(reflect.TypeFor[call.WireError]())
	head := func(keys map[string]any, required ...string) map[string]any {
		props := map[string]any{"v": map[string]any{"const": 2}, "id": map[string]any{"type": "string"}}
		maps.Copy(props, keys)
		return closed(props, slices.Concat([]string{"v", "id"}, required))
	}
	ref := func(key string) map[string]any { return map[string]any{"$ref": "#/$defs/" + key} }
	name := map[string]any{"type": "string"}
	r.defs["HelloFrame"] = head(map[string]any{"hello": ref("Hello")}, "hello")
	r.defs["CallFrame"] = head(map[string]any{"call": name, "target": ref("Target"), "in": map[string]any{}}, "call")
	r.defs["WatchFrame"] = head(map[string]any{"watch": name, "target": ref("Target"), "in": map[string]any{}}, "watch")
	r.defs["ProgressFrame"] = head(map[string]any{"progress": ref("Progress")}, "progress")
	r.defs["OutFrame"] = head(map[string]any{"out": map[string]any{}}, "out")
	r.defs["ErrorFrame"] = head(map[string]any{"error": wireError}, "error")
	r.defs["DoneFrame"] = head(map[string]any{"done": map[string]any{"const": true}}, "done")
	r.defs["Request"] = map[string]any{"anyOf": []any{ref("HelloFrame"), ref("CallFrame"), ref("WatchFrame")}}
	r.defs["Answer"] = map[string]any{"anyOf": []any{ref("HelloFrame"), ref("ProgressFrame"), ref("OutFrame"), ref("ErrorFrame"), ref("DoneFrame")}}
	return map[string]any{
		"$schema":     draft,
		"$id":         "https://github.com/jterrazz/studio-station/blob/main/" + framePath,
		"title":       "station.api/v2",
		"description": "One frame of station.api/v2, one JSON object per line: a Request, or an Answer carrying the request's id. A call's `in` and `out` are its own schema's, under calls/. Written by make generate; remote/api's test holds it to remote/api.Frame.",
		"anyOf":       []any{ref("Request"), ref("Answer")},
		"$defs":       r.defs,
	}
}

func eventSchema(w *wireRules) map[string]any {
	r := &reflector{defs: map[string]any{}, wire: w}
	all := make([]any, 0, len(events()))
	for _, e := range events() {
		all = append(all, r.event(e))
	}
	r.defs["Event"] = map[string]any{"anyOf": all}
	opened := r.wireOf(reflect.TypeFor[engine.Opened]())
	return map[string]any{
		"$schema":     draft,
		"$id":         "https://github.com/jterrazz/studio-station/blob/main/" + eventPath,
		"title":       "station.api/v2 events",
		"description": "An `out` of the watches page and engine.events: their opening, then an event, told apart by `event`. Written by make generate from engine.Event and engine.Opened.",
		"anyOf":       []any{opened, map[string]any{"$ref": "#/$defs/Event"}},
		"$defs":       r.defs,
	}
}

// value: the generated files of packages/api, by their path from the root.
func apiFiles(root string) map[string][]byte {
	w := newWireRules(root)
	calls := wireCalls()
	w.learnAnswers(calls)
	out := map[string][]byte{}
	docs := map[string]map[string]any{}
	for _, c := range calls {
		doc := callSchema(w, c)
		docs[c.meta.Name] = doc
		out[filepath.Join(apiCallsDir, c.meta.Name+".json")] = indented(doc)
	}
	frame, event := frameSchema(w), eventSchema(w)
	out[framePath] = indented(frame)
	out[eventPath] = indented(event)
	out[typesPath] = typescript(calls, docs, frame, event)
	return out
}

// invariant: a call's schema left behind by a call removed is not deleted
// here (cmd/generate writes and never removes): the drift test names it.
func writeAPI(root string) error {
	if err := os.MkdirAll(filepath.Join(root, apiCallsDir), 0o750); err != nil {
		return err
	}
	files := apiFiles(root)
	for _, path := range slices.Sorted(maps.Keys(files)) {
		if err := os.WriteFile(filepath.Join(root, path), files[path], 0o600); err != nil {
			return err
		}
	}
	return nil
}
