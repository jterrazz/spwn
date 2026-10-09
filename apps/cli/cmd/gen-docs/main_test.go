package main

import (
	"os"
	"testing"

	cli "spwn.sh/apps/cli"
)

// The committed page is what the generator writes for the current tree, so a
// command whose help changed without `make docs` fails here, not in a reader's
// browser.
func TestCommandsPageIsFresh(t *testing.T) {
	committed, err := os.ReadFile("../../../../docs/reference/commands.md")
	if err != nil {
		t.Fatalf("read the committed page: %v", err)
	}
	if got := render(cli.GetRootCmd()); string(got) != string(committed) {
		t.Fatalf("docs/reference/commands.md is stale: run `make docs`")
	}
}
