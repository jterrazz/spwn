package calls

import (
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type loggedControl struct {
	Control
	log engine.Log
}

func (c loggedControl) Log(context.Context) (engine.Log, error) { return c.log, nil }

func TestALineOfTheEngineLogLeavesWithItsSecretWithheld(t *testing.T) {
	d := Deps{Control: loggedControl{log: engine.Log{Lines: []string{"ping https://u:hunter2@example.com/x ok"}}}}
	got, err := ReadLog.Run(context.Background(), d, LogIn{}, nil)
	if err != nil || len(got.Lines) != 1 || strings.Contains(got.Lines[0], "hunter2") {
		t.Errorf("log = %+v, %v; want the password withheld", got, err)
	}
}
