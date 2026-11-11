package calls

import (
	"context"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
)

const ReadBudget = 5 * time.Second

// invariant: an item runs through the Applier of its effect's kind; an
// effect no Applier writes is an error, never a silent success.
func (s *Catalogue) handler(it catalog.Setting) (handler, error) {
	if p, ok := it.Effect.(catalog.Procedure); ok && !p.Known() {
		return handler{}, &settings.UnhandledError{Item: it.Name, Builtin: string(p)}
	}
	a, ok := s.appliers.For(it)
	if !ok {
		return handler{}, &settings.UnhandledError{Item: it.Name}
	}
	h := handler{
		install: func(ctx context.Context, values map[string]string) (settings.Change, error) {
			return a.Apply(ctx, it, values)
		},
	}
	if !it.RunOnce {
		h.check = func(ctx context.Context) (catalog.State, string) {
			o, err := a.Observe(ctx, it)
			if err != nil {
				return catalog.StateBroken, firstLine(err.Error())
			}
			return o.State, o.Detail
		}
	}
	if it.Reversible {
		h.uninstall = func(ctx context.Context) (settings.Change, error) { return a.Undo(ctx, it) }
	}
	return h, nil
}

func (s *Catalogue) Appliers() settingscontract.Appliers { return s.appliers }

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
