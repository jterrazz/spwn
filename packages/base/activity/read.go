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

	"spwn.sh/packages/base"
)

// ReadOpts filters the returned events.
type ReadOpts struct {
	Limit   int       // 0 = no limit
	Type    Type      // "" = any type
	WorldID string    // "" = any world
	AgentID string    // "" = any agent
	Actor   string    // "" = any actor
	Since   time.Time // zero = no lower bound
}

// Read returns events matching the filter, newest first.
func Read(opts ReadOpts) ([]Event, error) {
	path := base.ActivityPath()
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
	// Allow longer lines for rich metadata
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		if !matchesFilter(e, opts) {
			continue
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return events, err
	}

	// Sort newest first
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
