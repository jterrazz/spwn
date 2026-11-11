package appsmove

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Input struct {
	Name string
	To   string
}

func binder(managers []string) func(in *Input, args []string) error {
	return func(in *Input, args []string) error {
		in.Name = args[0]
		switch {
		case in.To == "":
			return kit.Refuse("give the manager it moves to", "--to takes "+strings.Join(managers, ", "))
		case !slices.Contains(managers, in.To):
			return kit.Refuse("unknown manager \""+in.To+"\"", "--to takes "+strings.Join(managers, ", "))
		}
		return nil
	}
}
