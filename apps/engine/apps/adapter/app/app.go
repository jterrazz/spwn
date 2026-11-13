package app

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type App struct{}

func New() App { return App{} }

func (App) Method() catalog.Method    { return catalog.MethodApp }
func (App) Manager() contract.Manager { return contract.Manager{} }
func (App) Can() contract.Abilities   { return contract.Abilities{} }

func (App) Ready(context.Context) error { return nil }

func (App) Inventory(context.Context) (map[string]string, error) {
	return nil, contract.ErrUnlisted
}

func (App) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (App) Install(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}

func (App) Update(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}

func (App) Remove(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}
