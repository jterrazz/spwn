package calls_test

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// value: a Mac with no files, whose clock moves only by what is slept; a
// sleep first calls pause, when set, with how long it was asked for.
type stillMac struct {
	mu    sync.Mutex
	at    time.Time
	pause func(d time.Duration)
}

func (m *stillMac) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.at
}

func (m *stillMac) Sleep(_ context.Context, d time.Duration) error {
	if m.pause != nil {
		m.pause(d)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.at = m.at.Add(d)
	return nil
}
func (*stillMac) Home() string                   { return "/home" }
func (*stillMac) UID() int                       { return 501 }
func (*stillMac) User() string                   { return "owner" }
func (*stillMac) Hostname() string               { return "mac" }
func (*stillMac) Exists(string) bool             { return false }
func (*stillMac) Names(string) ([]string, error) { return nil, nil }
func (*stillMac) Remove(string) error            { return nil }

// value: preferences kept in memory, each group as its JSON.
type memoryPrefs map[string][]byte

func (p memoryPrefs) Read(group string, v any) error {
	if data, ok := p[group]; ok {
		return json.Unmarshal(data, v)
	}
	return nil
}

func (p memoryPrefs) Write(group string, v any) error {
	data, err := json.Marshal(v)
	p[group] = data
	return err
}
