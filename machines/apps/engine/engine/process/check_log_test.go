package process

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type refusedCheck struct{}

func (refusedCheck) Run(context.Context, model.CheckTarget) (model.CheckAnswer, error) {
	return model.CheckAnswer{}, errors.New("503")
}

func TestAFailingChecksLogLineWithholdsItsToken(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.AlwaysOn, Check: "https://svc.example/health?token=" + token})
	h.d.o.Checks = refusedCheck{}
	notes := make(chan string, 16)
	h.d.note = func(line string) { notes <- line }
	v, _ := h.d.find("api")
	if err := h.d.start(cancelled(t), v); err != nil {
		t.Fatal(err)
	}
	h.d.mu.Lock()
	v.ip, v.state, v.check = "192.168.64.9", engine.StateRunning, model.CheckStatus{State: model.CheckPassing}
	h.d.mu.Unlock()
	h.d.keep(t.Context(), v)
	wait, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	for {
		select {
		case line := <-notes:
			if !strings.Contains(line, "its check fails") {
				continue
			}
			if strings.Contains(line, token) || !strings.HasSuffix(line, "its check fails: https://svc.example/health?token=withheld") {
				t.Errorf("daemon.log: %q", line)
			}
			return
		case <-wait.Done():
			t.Fatal("no line said the check fails")
		}
	}
}
