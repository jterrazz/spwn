package setup

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type Step struct {
	Name    string
	State   catalog.State
	Detail  string
	Log     string
	Changed bool
}

type Report struct{ Steps []Step }
