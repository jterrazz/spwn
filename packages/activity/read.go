package activity

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"spwn.sh/packages/platform"
)

// ReadOpts filters the returned events.
type ReadOpts struct {
	Limit   int
	Type    Type
	WorldID string
	AgentID string
	Actor   string
	Since   time.Time
}

// Read returns events matching the filter, newest first.
func Read(opts ReadOpts) ([]Event, error) {
	path := platform.ActivityPath()
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Event{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if !matchesFilter(e, opts) {
			continue
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return events, err
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}
	return events, nil
}

func matchesFilter(e Event, opts ReadOpts) bool {
	if opts.Type != "" && e.Type != opts.Type {
		return false
	}
	if opts.WorldID != "" && e.WorldID != opts.WorldID {
		return false
	}
	if opts.AgentID != "" && e.AgentID != opts.AgentID {
		return false
	}
	if opts.Actor != "" && e.Actor != opts.Actor {
		return false
	}
	if !opts.Since.IsZero() && e.Timestamp.Before(opts.Since) {
		return false
	}
	return true
}
