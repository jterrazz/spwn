package catalog

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// invariant: a value decoded by its own UnmarshalYAML is read as strictly
// as the file: every key of a mapping is one of its type's, refused with
// its line as the strict decoder refuses one.
func knownKeys(n *yaml.Node, t reflect.Type, extra ...string) error {
	names := slices.Clone(extra)
	for f := range t.Fields() {
		if name, _, _ := strings.Cut(f.Tag.Get("yaml"), ","); name != "" && name != "-" {
			names = append(names, name)
		}
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if k := n.Content[i]; !slices.Contains(names, k.Value) {
			return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: field %s not found in type %s", k.Line, k.Value, t)}}
		}
	}
	return nil
}

func (a *AppEntry) UnmarshalYAML(n *yaml.Node) error {
	// invariant: an app is written as its name, or as a mapping.
	if n.Kind == yaml.ScalarNode {
		*a = AppEntry{Name: n.Value, Line: n.Line}
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return errAppEntry
	}
	type plain AppEntry
	if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
		return err
	}
	if err := n.Decode((*plain)(a)); err != nil {
		return err
	}
	a.Line = n.Line
	return nil
}

func (AppEntry) JSONSchema() map[string]any {
	scalar := OptionValue("").JSONSchema()
	entry := map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"name"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string"}, "version": map[string]any{"type": []string{"string", "number"}, "description": "how far the app moves: " + RangeForms},
			"optional": map[string]any{"type": "boolean"}, "manager": map[string]any{"type": "string", "enum": ManagerKey("").Enum()},
			"options": map[string]any{"type": "object", "additionalProperties": scalar},
		},
	}
	return map[string]any{"oneOf": []any{map[string]any{"type": "string"}, entry}}
}

var (
	errAppEntry    = errors.New("an app is its name, or a mapping with its name")
	errAppList     = errors.New("apps is a list; an overlay's is a mapping of add and remove")
	errSettingList = errors.New("settings is a mapping of setting names; an overlay's, of set and remove")
	errPorts       = errors.New("open is a port, a range (`3000-9999`), or a list of them")
)

func (l *AppList) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a sequence is a Machinefile's list; a mapping is an
	// overlay's changes, add and remove.
	switch n.Kind {
	case yaml.SequenceNode:
		return n.Decode(&l.List)
	case yaml.MappingNode:
		type changes struct {
			Add    []AppEntry `yaml:"add,omitempty"`
			Remove []string   `yaml:"remove,omitempty"`
		}
		if err := knownKeys(n, reflect.TypeFor[changes]()); err != nil {
			return err
		}
		var c changes
		if err := n.Decode(&c); err != nil {
			return err
		}
		*l = AppList{Add: c.Add, Remove: c.Remove, Changes: true}
		return nil
	default:
		return errAppList
	}
}

func (AppList) JSONSchema() map[string]any {
	entry := AppEntry{}.JSONSchema()
	changes := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"add":    map[string]any{"type": "array", "items": entry},
			"remove": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
	return map[string]any{"oneOf": []any{map[string]any{"type": "array", "items": entry}, changes}}
}

func (s *SettingEntry) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a setting's mapping holds its option values, by option,
	// beside the two keys of the Machinefile's own, from and optional.
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a setting is a mapping of its option values (`{}` for none)", n.Line)
	}
	*s = SettingEntry{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			return fmt.Errorf("line %d: %s is a value: a string, an integer or a boolean", v.Line, k.Value)
		}
		switch k.Value {
		case "from":
			s.From = v.Value
		case "optional":
			optional, err := strconv.ParseBool(v.Value)
			if err != nil || v.ShortTag() != "!!bool" {
				return fmt.Errorf("line %d: optional is true or false", v.Line)
			}
			s.Optional = optional
		default:
			if s.Values == nil {
				s.Values = map[string]OptionValue{}
			}
			s.Values[k.Value] = OptionValue(v.Value)
		}
	}
	return nil
}

func (SettingEntry) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": OptionValue("").JSONSchema(),
		"properties": map[string]any{
			"from":     map[string]any{"type": "string", "enum": []string{FromComputer}},
			"optional": map[string]any{"type": "boolean"},
		},
	}
}

// value: the two keys of an overlay's settings.
var settingChanges = []string{"set", "remove"}

func (l *SettingList) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a mapping whose keys are set and remove alone is an
	// overlay's changes; any other, a Machinefile's settings by name (no
	// setting is named set or remove).
	if n.Kind != yaml.MappingNode {
		return errSettingList
	}
	changes := len(n.Content) > 0
	for i := 0; i < len(n.Content); i += 2 {
		changes = changes && slices.Contains(settingChanges, n.Content[i].Value)
	}
	if !changes {
		*l = SettingList{}
		if err := n.Decode(&l.Set); err != nil {
			return err
		}
		lines(n, l.Set)
		return nil
	}
	var c struct {
		Set    map[string]SettingEntry `yaml:"set,omitempty"`
		Remove []string                `yaml:"remove,omitempty"`
	}
	if err := n.Decode(&c); err != nil {
		return err
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == "set" {
			lines(n.Content[i+1], c.Set)
		}
	}
	*l = SettingList{Set: c.Set, Remove: c.Remove, Changes: true}
	return nil
}

// value: each setting of a mapping given the line its name is at.
func lines(n *yaml.Node, set map[string]SettingEntry) {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if e, ok := set[n.Content[i].Value]; ok {
			e.Line = n.Content[i].Line
			set[n.Content[i].Value] = e
		}
	}
}

func (SettingList) JSONSchema() map[string]any {
	entry := SettingEntry{}.JSONSchema()
	return map[string]any{
		"type": "object", "additionalProperties": entry,
		"properties": map[string]any{
			"set":    map[string]any{"type": "object", "additionalProperties": entry},
			"remove": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
}

func (p *PortSet) UnmarshalYAML(n *yaml.Node) error {
	// invariant: ports are one port or range, or a list of them; a range
	// is two ports joined by a dash, lowest first.
	nodes := []*yaml.Node{n}
	switch n.Kind {
	case yaml.SequenceNode:
		nodes = n.Content
	case yaml.ScalarNode:
	default:
		return errPorts
	}
	set := PortSet{}
	for _, c := range nodes {
		if c.Kind != yaml.ScalarNode {
			return errPorts
		}
		span, ok := portSpan(c.Value)
		if !ok {
			return fmt.Errorf("line %d: %q is no port nor range (`3000-9999`)", c.Line, c.Value)
		}
		set = append(set, span)
	}
	*p = set
	return nil
}

func portSpan(word string) (PortSpan, bool) {
	from, to, ranged := strings.Cut(word, "-")
	a, err := strconv.Atoi(strings.TrimSpace(from))
	if err != nil {
		return PortSpan{}, false
	}
	b := a
	if ranged {
		if b, err = strconv.Atoi(strings.TrimSpace(to)); err != nil {
			return PortSpan{}, false
		}
	}
	return PortSpan{From: a, To: b}, true
}

func (s PortSpan) String() string {
	if s.From == s.To {
		return strconv.Itoa(s.From)
	}
	return strconv.Itoa(s.From) + "-" + strconv.Itoa(s.To)
}

func (PortSet) JSONSchema() map[string]any {
	one := map[string]any{"type": []any{"integer", "string"}}
	return map[string]any{"oneOf": []any{one, map[string]any{"type": "array", "items": one, "minItems": 1}}}
}
