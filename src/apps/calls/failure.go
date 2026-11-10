package calls

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
)

// invariant: an installer's own words (its stderr, its exit) go to the log
// and only there (SPEC §12.3); the outcome says, in station's words, what
// failed and where the tool's output is kept.
func (s *Shelf) failed(verb string, list []catalog.App, cause error) string {
	// invariant: an interrupt is worded as one, and its cause is not the
	// installer's output, so the log keeps nothing of it.
	if errors.Is(cause, context.Canceled) {
		return apps.Interrupted
	}
	words := "the " + verb + " failed"
	if s.o.Log == "" || s.keep(verb, list, cause) != nil {
		return words
	}
	shown := s.o.Log
	if home := s.m.Home(); home != "" && strings.HasPrefix(shown, home+"/") {
		shown = "~" + strings.TrimPrefix(shown, home)
	}
	return words + ": its output is in " + shown
}

func (s *Shelf) keep(verb string, list []catalog.App, cause error) error {
	names := make([]string, 0, len(list))
	for _, t := range list {
		names = append(names, t.Name)
	}
	line := s.m.Now().UTC().Format(time.RFC3339) + " " + verb + " " + strings.Join(names, " ") + ": " + cause.Error() + "\n"
	return s.m.Keep(s.o.Log, line)
}
