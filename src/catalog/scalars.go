package catalog

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

func (r *Roles) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a kind is written as one word or as a list of them.
	if n.Kind == yaml.ScalarNode {
		*r = Roles{MachineRole(n.Value)}
		return nil
	}
	var list []MachineRole
	if err := n.Decode(&list); err != nil {
		return err
	}
	*r = list
	return nil
}

func (Roles) JSONSchema() map[string]any {
	one := map[string]any{"type": "string", "enum": MachineRole("").Enum()}
	return map[string]any{"oneOf": []any{one, map[string]any{"type": "array", "items": one, "minItems": 1}}}
}

var errNotScalar = errors.New("a value is a string, an integer or a boolean")

func (v *OptionValue) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a value is kept as its text, whatever its scalar type.
	if n.Kind != yaml.ScalarNode {
		return errNotScalar
	}
	*v = OptionValue(n.Value)
	return nil
}

func (OptionValue) JSONSchema() map[string]any {
	return map[string]any{"type": []any{"string", "integer", "boolean"}}
}
