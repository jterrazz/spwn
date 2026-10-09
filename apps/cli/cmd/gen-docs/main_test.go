package main

import (
	"os"
	"sort"
	"strings"
	"testing"

	cli "spwn.sh/apps/cli"

	"github.com/spf13/cobra"
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

// The cli specification files a session under the words of its command, and
// the checker reads that list from the specification's source; a word the
// list missed would file its sessions under the wrong subject with no error.
func TestSpecificationNamesEveryCommandWord(t *testing.T) {
	source, err := os.ReadFile("../../../../specs/cli/cli.specification.ts")
	if err != nil {
		t.Fatalf("read the cli specification: %v", err)
	}
	declared := declaredCommandWords(t, string(source))

	tree := map[string]bool{}
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range available(cmd) {
			tree[child.Name()] = true
			walk(child)
		}
	}
	walk(cli.GetRootCmd())

	want := sortedKeys(tree)
	if got := sortedKeys(declared); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("specs/cli/cli.specification.ts `commands` differs from the Cobra tree\n got: %v\nwant: %v", got, want)
	}
}

func declaredCommandWords(t *testing.T, source string) map[string]bool {
	t.Helper()
	open := strings.Index(source, "commands: [")
	if open < 0 {
		t.Fatalf("specs/cli/cli.specification.ts declares no `commands: [` list")
	}
	list := source[open+len("commands: ["):]
	list = list[:strings.Index(list, "]")]
	words := map[string]bool{}
	for _, entry := range strings.Split(list, ",") {
		if word := strings.Trim(strings.TrimSpace(entry), "'"); word != "" {
			words[word] = true
		}
	}
	return words
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
