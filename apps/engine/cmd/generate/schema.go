package main

import (
	"encoding"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

const (
	schemaPath            = "packages/api/schemas/station.package.v1.json"
	machinefileSchemaPath = "packages/api/schemas/station.machinefile.v1.json"
)

type enumerated interface{ Enum() []string }

// value: a type the decoder reads by its own UnmarshalYAML, which says its
// schema itself (a kind written as one word or a list).
type schematic interface{ JSONSchema() map[string]any }

// invariant: the JSON Schema of a file is projected from the Go types the
// decoder reads, so an editor's check and station's own refusal agree: a
// key without omitempty is required, every struct refuses a key it does not
// name, and a type with an Enum lists its values.
func catalogSchema() []byte {
	return projected(reflect.TypeFor[catalog.Package](), "package", catalog.Schema, schemaPath,
		"One package of station's catalog: a manager, an app or a setting, its methods, and an app's disk row. Written by make generate from apps/engine/catalog; each method names exactly one manager key. The format is docs/reference/catalog.md.")
}

func machinefileSchema() []byte {
	return projected(reflect.TypeFor[catalog.Machinefile](), "machinefile", catalog.MachinefileSchema, machinefileSchemaPath,
		"One Machinefile: what a machine carries (machine, vm, extends, prefer, apps, settings, accounts, ports), or its overlay <name>.local.yaml (overrides). Written by make generate from apps/engine/catalog; station alone checks what needs the catalog. The format is docs/reference/machinefiles.md.")
}

func projected(t reflect.Type, key, schema, at, description string) []byte {
	r := &reflector{defs: map[string]any{}}
	root := r.objectOf(t)
	if props, ok := root["properties"].(map[string]any); ok {
		props[key] = map[string]any{"const": schema}
	}
	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://github.com/jterrazz/studio-station/blob/main/" + at,
		"title":       schema,
		"description": description,
		"$defs":       r.defs,
	}
	for k, v := range root {
		doc[k] = v
	}
	return indented(doc)
}

func indented(doc any) []byte {
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(body, '\n')
}

// invariant: one reflector reads both encodings. Its YAML form is a file's,
// the decoder's strict reading; its wire form (wire non-nil) is what
// call.Encode writes and encoding/json reads: a key is the json tag or the field's name,
// an embedded struct's keys are its parent's, and a value the encoder may
// write as null (a nil slice, map, pointer or interface without omitempty) is
// said to be nullable.
type reflector struct {
	defs map[string]any
	wire *wireRules
	// value: the reading of a call's input: encoding/json fills a key it
	// finds and leaves the others zero, so none is required, and a client
	// never needs to send null.
	input bool
}

func (r *reflector) objectOf(t reflect.Type) map[string]any {
	if r.wire != nil {
		return r.wireObject(t)
	}
	props := map[string]any{}
	required := []string{}
	for f := range t.Fields() {
		name, opts, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "" && opts == "inline" {
			inlined := r.objectOf(f.Type)
			inner, _ := inlined["properties"].(map[string]any)
			maps.Copy(props, inner)
			if req, ok := inlined["required"].([]string); ok {
				required = append(required, req...)
			}
			continue
		}
		if name == "" || name == "-" {
			continue
		}
		props[name] = r.schemaOf(f.Type)
		if !slices.Contains(strings.Split(opts, ","), "omitempty") {
			required = append(required, name)
		}
	}
	return closed(props, required)
}

func closed(props map[string]any, required []string) map[string]any {
	out := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func (r *reflector) schemaOf(t reflect.Type) map[string]any {
	if r.wire != nil {
		return r.wireOf(t)
	}
	if s, ok := reflect.Zero(t).Interface().(schematic); ok && t.Kind() != reflect.Pointer {
		return s.JSONSchema()
	}
	if e, ok := reflect.Zero(t).Interface().(enumerated); ok && t.Kind() == reflect.String {
		return map[string]any{"type": "string", "enum": e.Enum()}
	}
	scalars := map[reflect.Kind]string{reflect.String: "string", reflect.Bool: "boolean", reflect.Int: "integer", reflect.Int64: "integer", reflect.Uint32: "integer"}
	if name, ok := scalars[t.Kind()]; ok {
		return map[string]any{"type": name}
	}
	if t.Kind() == reflect.Pointer {
		return r.schemaOf(t.Elem())
	}
	if t.Kind() == reflect.Slice {
		return map[string]any{"type": "array", "items": r.schemaOf(t.Elem())}
	}
	if t.Kind() == reflect.Map {
		return map[string]any{"type": "object", "additionalProperties": r.schemaOf(t.Elem())}
	}
	if t.Kind() == reflect.Struct {
		if _, seen := r.defs[t.Name()]; !seen {
			r.defs[t.Name()] = true
			r.defs[t.Name()] = r.objectOf(t)
		}
		return map[string]any{"$ref": "#/$defs/" + t.Name()}
	}
	panic("a schema has no JSON type for " + t.String())
}

var (
	jsonMarshaler = reflect.TypeFor[json.Marshaler]()
	textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
	errorType     = reflect.TypeFor[error]()
)

func marshals(t, as reflect.Type) bool {
	return t.Implements(as) || reflect.PointerTo(t).Implements(as)
}

func (r *reflector) wireOf(t reflect.Type) map[string]any {
	if s, ok := r.wire.custom[t]; ok {
		return s(r)
	}
	if t.Kind() == reflect.Pointer {
		return r.wireOf(t.Elem())
	}
	if marshals(t, jsonMarshaler) {
		panic("the wire schema has no rule for " + t.String() + ", which writes itself with MarshalJSON: name its form in wireRules.custom")
	}
	if marshals(t, textMarshaler) {
		return map[string]any{"type": "string"}
	}
	if t.Kind() == reflect.String {
		if e, ok := reflect.Zero(t).Interface().(enumerated); ok {
			return map[string]any{"type": "string", "enum": e.Enum()}
		}
		if values := r.wire.enums[enumKey(t)]; len(values) > 0 {
			return map[string]any{"type": "string", "enum": values}
		}
		return map[string]any{"type": "string"}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Interface:
		if t == errorType {
			return r.wireOf(reflect.TypeFor[call.WireError]())
		}
		if t.NumMethod() == 0 {
			return map[string]any{}
		}
		panic("the wire schema has no JSON type for the interface " + t.String())
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 && !marshals(t.Elem(), jsonMarshaler) && !marshals(t.Elem(), textMarshaler) {
			return map[string]any{"type": "string", "contentEncoding": "base64"}
		}
		return map[string]any{"type": "array", "items": r.nullable(t.Elem(), r.wireOf(t.Elem()))}
	case reflect.Array:
		return map[string]any{"type": "array", "items": r.nullable(t.Elem(), r.zeroable(t.Elem(), r.wireOf(t.Elem()))), "minItems": t.Len(), "maxItems": t.Len()}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": r.nullable(t.Elem(), r.wireOf(t.Elem()))}
	case reflect.Struct:
		if t.Name() == "" {
			return r.wireObject(t)
		}
		key := r.wire.key(t, r.input)
		if _, seen := r.defs[key]; !seen {
			r.defs[key] = true
			r.defs[key] = r.wireObject(t)
		}
		return map[string]any{"$ref": "#/$defs/" + key}
	default:
		panic("the wire schema has no JSON type for " + t.String())
	}
}

// invariant: an enum's zero, "", is one of its values wherever the encoder
// writes a zero it was never given: a key it does not omit, an array's
// element, a call's whole answer. An input never needs it, since an absent
// key reads as the same zero.
func (r *reflector) zeroable(t reflect.Type, s map[string]any) map[string]any {
	values, ok := s["enum"].([]string)
	if r.input || !ok || t.Kind() != reflect.String || slices.Contains(values, "") {
		return s
	}
	out := maps.Clone(s)
	out["enum"] = append([]string{""}, values...)
	return out
}

// invariant: what call.Encode writes for a value of t that is not
// omitted: null for a nil slice, map, pointer or interface. An input is
// never sent as null, since an absent key reads as the same zero.
func (r *reflector) nullable(t reflect.Type, s map[string]any) map[string]any {
	if r.input || r.wire.custom[t] != nil {
		return s
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Map, reflect.Pointer, reflect.Interface:
	default:
		return s
	}
	if len(s) == 0 {
		return s
	}
	if typ, ok := s["type"].(string); ok && len(s) <= 3 {
		if _, ref := s["$ref"]; !ref {
			out := maps.Clone(s)
			out["type"] = []any{typ, "null"}
			return out
		}
	}
	return map[string]any{"anyOf": []any{s, map[string]any{"type": "null"}}}
}

type wireField struct {
	name     string
	t        reflect.Type
	optional bool
	quoted   bool
}

// invariant: the keys call.Encode writes for a struct: its own walk's for a
// struct that holds an error (call.EncodedKeys), encoding/json's for any
// other, and encoding/json's for an input, which it decodes.
func encodedFields(t reflect.Type, input bool) []wireField {
	if keys, ok := call.EncodedKeys(t); ok && !input {
		out := make([]wireField, 0, len(keys))
		for _, k := range keys {
			out = append(out, wireField{name: k.Name, t: k.Type, optional: k.OmitEmpty})
		}
		return out
	}
	return wireFields(t)
}

// invariant: omitempty leaves out only a zero bool or number, an empty
// string, slice, map or array, a nil pointer or interface: a struct under
// omitempty is always written.
func omitsEmpty(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		return false
	case reflect.Array:
		return t.Len() == 0
	default:
		return true
	}
}

// invariant: the keys encoding/json gives a struct: an exported field by its
// tag or its name, `-` dropped, an untagged embedded struct (or pointer to
// one) flattened into its parent, the shallower of two equal names winning.
func wireFields(t reflect.Type) []wireField {
	type found struct {
		wireField
		depth  int
		tagged bool
	}
	var all []found
	var walk func(t reflect.Type, depth int, optional bool)
	walk = func(t reflect.Type, depth int, optional bool) {
		for f := range t.Fields() {
			tag, hasTag := f.Tag.Lookup("json")
			name, opts, _ := strings.Cut(tag, ",")
			if tag == "-" {
				continue
			}
			ft := f.Type
			if f.Anonymous && name == "" {
				inner := ft
				if inner.Kind() == reflect.Pointer {
					inner = inner.Elem()
				}
				if inner.Kind() == reflect.Struct {
					walk(inner, depth+1, optional || ft.Kind() == reflect.Pointer)
					continue
				}
			}
			if !f.IsExported() {
				continue
			}
			if name == "" {
				name = f.Name
			}
			options := strings.Split(opts, ",")
			omit := slices.Contains(options, "omitzero") || slices.Contains(options, "omitempty") && omitsEmpty(ft)
			all = append(all, found{wireField{name: name, t: ft, optional: optional || omit, quoted: slices.Contains(options, "string")}, depth, hasTag && name != ""})
		}
	}
	walk(t, 0, false)
	var out []wireField
	for _, f := range all {
		shadowed := false
		for _, g := range all {
			if g.name == f.name && (g.depth < f.depth || g.depth == f.depth && g.tagged && !f.tagged) {
				shadowed = true
			}
		}
		if !shadowed {
			out = append(out, f.wireField)
		}
	}
	return out
}

func (r *reflector) wireObject(t reflect.Type) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, f := range encodedFields(t, r.input) {
		var s map[string]any
		if over, ok := r.wire.fields[t.String()+"."+f.name]; ok {
			s = over(r)
		} else {
			s = r.wireOf(f.t)
			if f.quoted && (s["type"] == "integer" || s["type"] == "number" || s["type"] == "boolean") {
				s = map[string]any{"type": "string"}
			}
		}
		if f.optional || r.input {
			props[f.name] = s
			continue
		}
		props[f.name] = r.nullable(f.t, r.zeroable(f.t, s))
		required = append(required, f.name)
	}
	slices.Sort(required)
	return closed(props, required)
}

func enumKey(t reflect.Type) string {
	if t.PkgPath() == "" {
		return ""
	}
	return qualified(t)
}
