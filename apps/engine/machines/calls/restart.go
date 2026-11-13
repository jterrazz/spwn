package calls

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	pollEvery    = 2 * time.Second
	stopBudget   = 60 * time.Second
	returnBudget = 180 * time.Second
)

func (s *Machinist) Restart(ctx context.Context, name string, progress func(model.RestartEvent)) (model.RestartReport, error) {
	if err := s.CheckMac(); err != nil {
		return model.RestartReport{}, err
	}
	m, err := s.Remote(name)
	if err != nil {
		return model.RestartReport{}, err
	}
	say := func(phase, detail string) {
		if progress != nil {
			progress(model.RestartEvent{Phase: phase, Detail: detail})
		}
	}
	if m.Kind != catalog.Mac {
		return s.restartVM(ctx, name, say)
	}
	started := s.h.Now()

	say("restarting", "a planned restart that unlocks itself")
	var refused *contract.RestartRefusedError
	if err := s.o.Unlocker.PlannedRestart(ctx, m.Name); errors.As(err, &refused) {
		return model.RestartReport{}, &model.RestartRefusedError{Name: name, Detail: refused.Detail}
	} else if err != nil {
		return model.RestartReport{}, err
	}

	if err := s.await(ctx, m.Name, false, stopBudget); err != nil {
		return model.RestartReport{}, wrap(err, name, model.PhaseStop)
	}
	say("down", "the Mac is rebooting")
	if err := s.await(ctx, m.Name, true, returnBudget); err != nil {
		return model.RestartReport{}, wrap(err, name, model.PhaseReturn)
	}
	waited := s.h.Now().Sub(started)
	say("back", "the Mac answers on ssh again")
	return model.RestartReport{Name: name, Waited: waited}, nil
}

type budgetError struct{}

func (budgetError) Error() string { return "budget spent" }

func wrap(err error, name, phase string) error {
	if errors.Is(err, budgetError{}) {
		return &model.RestartTimeoutError{Name: name, Phase: phase}
	}
	return err
}

func (s *Machinist) await(ctx context.Context, name string, up bool, budget time.Duration) error {
	for waited := time.Duration(0); ; waited += pollEvery {
		if s.answers(ctx, name) == up {
			return nil
		}
		if waited >= budget {
			return budgetError{}
		}
		if err := s.h.Sleep(ctx, pollEvery); err != nil {
			return err
		}
	}
}

func (s *Machinist) AwaitAnswer(ctx context.Context, mac string) bool {
	return s.await(ctx, mac, true, returnBudget) == nil
}

func (s *Machinist) Answers(ctx context.Context, mac string) bool { return s.answers(ctx, mac) }

func (s *Machinist) answers(ctx context.Context, mac string) bool {
	return s.o.Answers != nil && s.o.Answers(ctx, mac)
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return "no detail"
}
