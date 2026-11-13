package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

func Password(t *testing.T, open func(t *testing.T, asks bool) contract.Password) {
	ctx := context.Background()
	for _, asks := range []bool{true, false} {
		t.Run(map[bool]string{true: "a machine that asks a password reads asked", false: "a machine that asks none reads not asked"}[asks], func(t *testing.T) {
			got, err := open(t, asks).Asked(ctx)
			if err != nil || got != asks {
				t.Errorf("Asked = %v, %v; want %v", got, err, asks)
			}
		})
	}
}
