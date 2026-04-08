package python

import "testing"

func TestPython_Name(t *testing.T) {
	if Tool.Name() != "@python" {
		t.Errorf("expected @python, got %s", Tool.Name())
	}
}

func TestPython_HasVerify(t *testing.T) {
	if len(Tool.Verify()) == 0 {
		t.Error("@python should have verify commands")
	}
}
