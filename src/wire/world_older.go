//go:build world

package wire

import (
	"encoding/json"
	"errors"

	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/remote/api"
)

// value: what the binary of a Mac of the world that runs version refuses
// to read, as a station older than api.WordsSince does: a workspace's
// lifetime in any word but its own, throwaway and permanent (ADR-032),
// and an update's manager in any word but its own (`skills`, `brew` for
// cask), which it would read as no manager; nil for a Mac that runs this
// binary's words.
func olderRefusal(version string) func(name string, in json.RawMessage) error {
	if !api.Older(version, Version()) {
		return nil
	}
	return func(name string, in json.RawMessage) error {
		if name == appscalls.Update.Name {
			return olderManagers(in)
		}
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

func olderManagers(in json.RawMessage) error {
	var update appscalls.UpdateIn
	_ = json.Unmarshal(in, &update)
	for _, word := range update.Method {
		if then, ok := catalog.OlderManager(catalog.ManagerOfOlder(word)); !ok || then != word {
			return errors.New("v3.0 has no manager " + word + ": it updates nothing")
		}
	}
	return nil
}
