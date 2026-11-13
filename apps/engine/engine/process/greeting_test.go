package process

import (
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func greets(d *Engine) bool {
	select {
	case <-d.Greeted():
		return false
	default:
		return true
	}
}

func TestOnlyTheSessionAVMHoldsEndsItsGreeting(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	h.d.adopt(cancelled(t))
	h.d.mu.Lock()
	v := h.d.vms["api"]
	v.gen = 2
	h.d.greetLocked(v)
	h.d.mu.Unlock()
	scan := []byte(`{"v":1,"type":"snapshot","ports":[]}`)
	h.d.fromSession(t.Context(), v, 1, scan)
	if !greets(h.d) {
		t.Fatal("the first scan of a session api no longer holds ended the greeting of the one it holds")
	}
	h.d.greetedBy(v, 1)
	if !greets(h.d) {
		t.Fatal("the end of a session api no longer holds ended the greeting of the one it holds")
	}
	h.d.fromSession(t.Context(), v, 2, scan)
	if greets(h.d) {
		t.Fatal("the session api holds sent its first scan and still greets")
	}
}
