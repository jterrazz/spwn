package calls

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// invariant: an installer's own words (its output, its exit) go to the log of
// each app it served, `<logs>/<app>.log`, and only there (SPEC §12.3); the
// outcome says, in station's words, what failed and which file keeps why.
func (s *Shelf) failed(verb string, list []catalog.App, cause error) (func(app string) (words, log string), apps.Why) {
	// invariant: an interrupt is a typed reason, worded by the surface, and
	// its cause is not the installer's output, so the log keeps nothing of it.
	if errors.Is(cause, context.Canceled) {
		return func(string) (string, string) { return "", "" }, apps.WhyInterrupted
	}
	words := "the " + verb + " failed"
	kept := map[string]bool{}
	for _, t := range list {
		kept[t.Name] = s.keep(verb, t.Name, cause) == nil
	}
	return func(app string) (string, string) {
		if !kept[app] {
			return words, ""
		}
		log := s.shown(s.logOf(app))
		return words + ": its output is in " + log, log
	}, apps.WhyFailing
}

func (s *Shelf) logs(list []catalog.App) contract.Logs {
	if s.o.Logs == "" {
		return nil
	}
	out := make(contract.Logs, 0, len(list))
	for _, t := range list {
		out = append(out, s.logOf(t.Name))
	}
	return out
}

func (s *Shelf) logOf(app string) string { return path.Join(s.o.Logs, app+".log") }

func (s *Shelf) shown(file string) string {
	if home := s.m.Home(); home != "" && strings.HasPrefix(file, home+"/") {
		return "~" + strings.TrimPrefix(file, home)
	}
	return file
}

func (s *Shelf) keep(verb, app string, cause error) error {
	if s.o.Logs == "" {
		return errors.New("no logs folder")
	}
	line := s.m.Now().UTC().Format(time.RFC3339) + " " + verb + " " + app + ": " + cause.Error() + "\n"
	return s.m.Keep(s.logOf(app), line)
}
