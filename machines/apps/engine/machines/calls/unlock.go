package calls

import (
	"context"
	"errors"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

func (s *Machinist) Unlock(ctx context.Context, name string) error {
	if err := s.CheckMac(); err != nil {
		return err
	}
	m, err := s.RemoteMac(name, "unlock")
	if err != nil {
		return err
	}
	if err := s.o.Unlocker.Unlock(ctx, m.Name, m.LAN); errors.Is(err, contract.ErrUnreachable) {
		return &model.UnreachableError{Name: name}
	} else if err != nil {
		return err
	}
	return nil
}
