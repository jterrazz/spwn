//go:build world

package wire

import (
	"encoding/json"

	model "github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/remote/api"
)

// value: what the binary of a Mac of the world that runs version refuses
// to read, as a station older than api.WordsSince does: a workspace's
// lifetime in any word but its own, throwaway and permanent (ADR-032); nil
// for a Mac that runs this binary's words.
func olderRefusal(version string) func(name string, in json.RawMessage) error {
	if !api.Older(version, Version()) {
		return nil
	}
	return func(name string, in json.RawMessage) error {
		if name != machinescalls.CreateWorkspace.Name {
			return nil
		}
		var create struct{ Request struct{ Lifetime string } }
		_ = json.Unmarshal(in, &create)
		if word := create.Request.Lifetime; word != "" && word != model.OlderEphemeral && word != model.OlderPersistent {
			return &model.BadLifetimeError{Value: word}
		}
		return nil
	}
}
