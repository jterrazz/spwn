package engine

import (
	"context"
	"strings"
	"time"
)

// value: the measures a page reads live (SPEC §B3), each keyed
// `<measure>:<machine>`: CPU's share, the memory used and free, the disk
// used and the network's rate, in and out together.
const (
	LiveCPU    = "cpu"
	LiveMemory = "memory"
	LiveFree   = "free"
	LiveDisk   = "disk"
	LiveNet    = "net"
)

func LiveKey(measure, machine string) string { return measure + ":" + machine }

func LiveOf(key string) (measure, machine string) {
	// value: the measure a live key reads and its machine; no machine for a
	// key of none.
	measure, machine, _ = strings.Cut(key, ":")
	return measure, machine
}

const (
	// value: the stream's frame of the moving numbers, one every 2 s while
	// a watch of pages lists live keys.
	EventSample = "sample"
)

const SampleEvery = 2 * time.Second

type Sample struct {
	// value: each live key's value, as the page that lists the key draws it.
	Values map[string]any `json:"values,omitempty"`
	// value: the keys whose machine has not answered for longer than its
	// freshness: their value is the last known, marked stale.
	Stale []string `json:"stale,omitempty"`
}

type Sampler func(ctx context.Context, keys []string) Sample // value: one tick's reading of the live keys the open pages list; it never waits on a late machine, whose last value it keeps.
