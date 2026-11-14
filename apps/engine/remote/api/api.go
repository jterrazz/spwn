package api

import (
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract"
)

const (
	Name    = "station.api/v2"
	Version = 2
)

type Target struct {
	Name string `json:"name,omitempty"`
	On   string `json:"on,omitempty"`
}

func TargetOf(r machine.Ref) *Target {
	if r == (machine.Ref{}) {
		return nil
	}
	return &Target{Name: r.Name, On: r.On}
}

func (t *Target) Ref() machine.Ref {
	if t == nil {
		return machine.Ref{}
	}
	return machine.Ref{Name: t.Name, On: t.On}
}

type Hello = contract.Hello

type Progress struct {
	Task   string          `json:"task"`
	Detail string          `json:"detail,omitempty"`
	Done   bool            `json:"done,omitempty"`
	Error  *call.WireError `json:"error,omitempty"`
}

type Frame struct {
	// invariant: one JSON object per line. A request carries call or watch
	// (or hello); its answers carry the same id: progress frames, then one
	// out or error for a call, out frames then done for a watch. A cancel
	// carries the id of a request still answering and ends it.
	V        int             `json:"v"`
	ID       string          `json:"id"`
	Hello    *Hello          `json:"hello,omitempty"`
	Call     string          `json:"call,omitempty"`
	Watch    string          `json:"watch,omitempty"`
	Target   *Target         `json:"target,omitempty"`
	In       json.RawMessage `json:"in,omitempty"`
	Progress *Progress       `json:"progress,omitempty"`
	Out      json.RawMessage `json:"out,omitempty"`
	Error    *call.WireError `json:"error,omitempty"`
	Done     bool            `json:"done,omitempty"`
	Cancel   bool            `json:"cancel,omitempty"`
}

type Down struct{}

func (*Down) Error() string { return "the station engine is not running" }
