package activitycmd

import (
	"errors"
	"strings"
	"testing"

	pageactivity "github.com/jterrazz/studio-station/apps/engine/pages/activity"
)

func TestAMachineNotReadSaysItsErrorWithItsTokenWithheld(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	v := silenced(pageactivity.View{Name: "linux-4"}, errors.New("fetch https://bot:hunter2@host/x refused: token "+token))
	if v.Silent != "fetch https://withheld@host/x refused: token withheld" {
		t.Errorf("silent = %q; want each secret withheld", v.Silent)
	}
}
