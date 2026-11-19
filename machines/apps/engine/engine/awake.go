package engine

import (
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/awake"
)

const (
	NoticeAwakeEnded  = "awake_ended"
	NoticeNeverSleeps = "never_sleeps"
)

type AwakeOff struct {
	Ended bool `json:"ended"`
	After int  `json:"after"`
}

type StopAll struct {
	VMs        []Stopped `json:"vms"`
	AwakeEnded bool      `json:"awake_ended"`
	// value: a port of a VM it stopped was shared, and is no more.
	SharingEnded bool `json:"sharing_ended,omitempty"`
}

type Notice struct {
	Kind string `json:"kind"`
	Icon string `json:"icon,omitempty"`
	Text string `json:"text"`
}

type Event struct {
	Event  string          `json:"event"`
	Page   json.RawMessage `json:"page,omitempty"`
	At     string          `json:"at,omitempty"`
	Notice *Notice         `json:"notice,omitempty"`
	Values map[string]any  `json:"values,omitempty"`
	Stale  []string        `json:"stale,omitempty"`
}

func AwakeStopped(detail string) string {
	// value: the words of SPEC §10.4 for Awake a guard stopped on the
	// engine's own Mac.
	return "Awake stopped on This Mac: " + detail
}

type AwakeKept = awake.Kept
