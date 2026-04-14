package plugin

import (
	"bytes"
	"strings"
	"testing"
)

func TestPluginLsCmd_RunsSuccessfully(t *testing.T) {
	// ls should always run without error — even when the registered
	// plugin list is empty, it prints a friendly message.
	var out bytes.Buffer
	lsCmd.SetOut(&out)
	lsCmd.SetErr(&out)
	if err := lsCmd.RunE(lsCmd, nil); err != nil {
		t.Fatalf("ls RunE: %v", err)
	}
	got := out.String()
	if got == "" {
		t.Error("ls produced no output")
	}
}

func TestPluginShowCmd_UnknownReturnsError(t *testing.T) {
	var out bytes.Buffer
	showCmd.SetOut(&out)
	showCmd.SetErr(&out)
	err := showCmd.RunE(showCmd, []string{"@spwn/does-not-exist"})
	if err == nil {
		t.Fatal("show on unknown plugin should error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}
