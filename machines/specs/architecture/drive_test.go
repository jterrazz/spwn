package architecture_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	command "github.com/spf13/cobra"
)

var updateDrive = flag.Bool("update-drive", false, "rewrite the spwn skill's drive section from the command tree")

// value: where the drive section of the machine's `spwn` skill lands,
// embedded in the binary by the catalog package.
const drivePath = "apps/engine/catalog/spwn/drive.txt"

// invariant: the drive section is the command tree's own words: each
// visible noun with its short line and each of its commands as it is used,
// and what spwn is, quoted from the two READMEs; `make generate` writes it,
// and any other run fails while it differs, so no line of it is ever
// written by hand.
func TestTheSpwnSkillsDriveSectionIsTheCommandTrees(t *testing.T) {
	root := repoRoot(t)
	want := driveSection(t, commandTree(t), root)
	path := filepath.Join(root, drivePath)
	if *updateDrive {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v: make generate writes it", drivePath, err)
	}
	if string(got) != want {
		t.Errorf("%s drifted from the command tree or the READMEs: make generate rewrites it", drivePath)
	}
	if again := driveSection(t, commandTree(t), root); again != want {
		t.Error("two builds of the drive section differ: it must be the same bytes from the same tree")
	}
}

func driveSection(t *testing.T, tree *command.Command, root string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("Spwn, in its README's words:\n\n")
	b.WriteString("> " + readmeParagraph(t, filepath.Join(root, "..", "README.md"), "Spwn is a stack") + "\n\n")
	b.WriteString("Machines, the space this machine is set up by, in its own README's words:\n\n")
	b.WriteString("> " + readmeParagraph(t, filepath.Join(root, "README.md"), "A single CLI") + "\n\n")
	name := tree.Name()
	b.WriteString("Its command here is `" + name + "`: `" + name + " <noun> [<verb>] [<name>...]`. `" + name + " help <command>` prints what a command does, its flags and examples; never guess a flag.\n")
	nouns := slices.DeleteFunc(slices.Clone(tree.Commands()), func(c *command.Command) bool { return c.Hidden || c.Name() == "help" || c.Name() == "completion" })
	for _, noun := range nouns {
		b.WriteString("\n- `" + name + " " + usage(noun) + "`: " + strings.TrimSuffix(noun.Short, ".") + ".")
		var verbs []string
		for _, c := range typedLeaves(noun) {
			verbs = append(verbs, "`"+strings.TrimSpace(strings.TrimPrefix(commandPath(c), noun.Name()))+suffix(c)+"`")
		}
		if len(verbs) > 0 {
			b.WriteString(" " + strings.Join(verbs, ", "))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// value: the visible commands under a noun that run, each with its path
// below it, in the tree's order.
func typedLeaves(noun *command.Command) []*command.Command {
	var out []*command.Command
	for _, c := range typedCommands(noun) {
		if c.Runnable() {
			out = append(out, c)
		}
	}
	return out
}

func usage(c *command.Command) string { return c.Name() + suffix(c) }

func suffix(c *command.Command) string {
	_, args, _ := strings.Cut(c.Use, " ")
	if args = strings.TrimSpace(args); args != "" {
		return " " + args
	}
	return ""
}

// value: the README's paragraph that opens with the words given, as one
// line.
func readmeParagraph(t *testing.T, path, opening string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, para := range strings.Split(string(body), "\n\n") {
		if strings.HasPrefix(para, opening) {
			return strings.Join(strings.Fields(para), " ")
		}
	}
	t.Fatalf("%s holds no paragraph opening %q", path, opening)
	return ""
}
