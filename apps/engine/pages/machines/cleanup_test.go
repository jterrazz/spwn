package machines

import (
	"slices"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestACleanupOfManyAsksOnceAndListsEachOnesLosses(t *testing.T) {
	cands := []model.Candidate{
		{Name: "agent-7f", DiskBytes: 12e9, Ports: []int{5173}, Accounts: []string{"GitHub", "Claude"}},
		{Name: "linux-3", DiskBytes: 20e9},
	}
	c := CleanupConfirm(cands)
	if c.Question != "Delete 2 ephemeral workspaces?" || !c.Danger || c.Yes != "Delete" {
		t.Fatalf("question %q danger %v yes %q", c.Question, c.Danger, c.Yes)
	}
	want := [][]string{
		{"agent-7f", "stopped", "goes with its disk (12 GB), 1 shared port and 2 grants"},
		{"linux-3", "stopped", "goes with its disk (20 GB)"},
	}
	if len(c.Lines) != len(want) {
		t.Fatalf("lines %v", c.Lines)
	}
	for i, w := range want {
		var got []string
		for _, v := range c.Lines[i].Items {
			got = append(got, v.Text)
		}
		if !slices.Equal(got, w) {
			t.Errorf("line %d: %q, want %q", i, got, w)
		}
	}
	if got := CleanupRefused(cands).Text; got != "refusing to delete 2 ephemeral workspaces without --yes: agent-7f goes with its disk (12 GB), 1 shared port and 2 grants; linux-3 goes with its disk (20 GB)" {
		t.Errorf("refusal %q", got)
	}
}
