package main

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	schemaPath            = "schemas/station.catalog.v1.json"
	machinefileSchemaPath = "schemas/station.machinefile.v1.json"
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
	return projected(reflect.TypeFor[catalog.Item](), "item", catalog.Schema, schemaPath,
		"One item of station's catalog: a manager, an app or a setting, its methods, and an app's disk row. Written by make generate from src/catalog; each method names exactly one manager key. The format is docs/reference/catalog.md.")
}

func machinefileSchema() []byte {
	return projected(reflect.TypeFor[catalog.Machinefile](), "machinefile", catalog.MachinefileSchema, machinefileSchemaPath,
		"One Machinefile: what a machine carries (machine, vm, extends, prefer, apps, settings, accounts, ports), or its overlay <name>.local.yaml (overrides). Written by make generate from src/catalog; station alone checks what needs the catalog. The format is docs/reference/machinefiles.md.")
}

func projected(t reflect.Type, key, schema, at, description string) []byte {
	defs := map[string]any{}
	root := objectOf(t, defs)
	if props, ok := root["properties"].(map[string]any); ok {
		props[key] = map[string]any{"const": schema}
	}
	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://github.com/jterrazz/studio-station/blob/main/" + at,
		"title":       schema,
		"description": description,
		"$defs":       defs,
	}
	for k, v := range root {
		doc[k] = v
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(body, '\n')
}

func objectOf(t reflect.Type, defs map[string]any) map[string]any {
	props := map[string]any{}
	required := []string{}
	for f := range t.Fields() {
		name, opts, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "" && opts == "inline" {
			inlined := objectOf(f.Type, defs)
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
		props[name] = schemaOf(f.Type, defs)
		if !slices.Contains(strings.Split(opts, ","), "omitempty") {
			required = append(required, name)
		}
	}
	out := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func schemaOf(t reflect.Type, defs map[string]any) map[string]any {
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
		return schemaOf(t.Elem(), defs)
	}
	if t.Kind() == reflect.Slice {
		return map[string]any{"type": "array", "items": schemaOf(t.Elem(), defs)}
	}
	if t.Kind() == reflect.Map {
		return map[string]any{"type": "object", "additionalProperties": schemaOf(t.Elem(), defs)}
	}
	if t.Kind() == reflect.Struct {
		if _, seen := defs[t.Name()]; !seen {
			defs[t.Name()] = true
			defs[t.Name()] = objectOf(t, defs)
		}
		return map[string]any{"$ref": "#/$defs/" + t.Name()}
	}
	panic("a schema has no JSON type for " + t.String())
}
