package settings

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type Report struct {
	Item    catalog.Setting
	State   catalog.State
	Detail  string
	Applies bool
}

type Change struct {
	Item   string
	Verb   string
	Result catalog.State
	Before []KeyValue
	After  []KeyValue
	Steps  []ChangeStep
}

type KeyValue struct{ Key, Value string }

type ChangeStep struct {
	Label  string
	State  catalog.State
	Detail string
}

type StudioMissingError struct{ Item string }

func (e *StudioMissingError) Error() string {
	return e.Item + " is kept by the studio CLI, which is not on PATH"
}

func Sets(it catalog.Setting) []KeyValue {
	// value: the values station sets for a setting that writes no file.
	out := make([]KeyValue, len(it.Sets))
	for i, v := range it.Sets {
		out[i] = KeyValue{Key: v.Key, Value: v.Value}
	}
	return out
}
