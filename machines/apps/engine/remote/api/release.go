package api

import "github.com/jterrazz/studio-station/apps/engine/machines"

// value: the rule a hello's version is judged by, machines.SkewOf, for the
// remote adapters, which speak this package and no family's model.

type Skew = machines.Skew

const (
	Speaks     = machines.Speaks
	FarBehind  = machines.FarBehind
	HereBehind = machines.HereBehind
)

func SkewOf(own, peer string) Skew { return machines.SkewOf(own, peer) }
