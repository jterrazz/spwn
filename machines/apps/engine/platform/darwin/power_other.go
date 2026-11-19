//go:build !darwin || !cgo

package darwin

import (
	"context"
	"errors"
)

const Supported = false

var errUnsupported = errors.New("no IOKit on this build")

type Reading struct {
	Lid        string
	Source     string
	HasBattery bool
	Percent    int
	Charging   bool
	LowPower   bool
	Thermal    int
	Health     int
	Cycles     int
}

type Power struct{}

func HoldIdle(string) (func(), error) { return nil, errUnsupported }

func SetClamshellSleep(bool) error { return errUnsupported }

func (*Power) Read() Reading { return Reading{} }

func SleepAnnounced() bool { return false }

func PowerSourcesChanged(context.Context) <-chan struct{} { return nil }
