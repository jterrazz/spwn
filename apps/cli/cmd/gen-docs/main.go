// Command gen-docs projects the Cobra command tree into one reference page,
// docs/reference/commands.md: every available command, depth first, its
// help text, its usage line and its flags.
package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	cli "spwn.sh/apps/cli"

	"github.com/spf13/cobra"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-docs <output-file>")
		os.Exit(1)
	}

	if err := os.WriteFile(os.Args[1], render(cli.GetRootCmd()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(root *cobra.Command) []byte {
	root.DisableAutoGenTag = true

	var page bytes.Buffer
	page.WriteString("# Commands\n\n")
	page.WriteString("Every command of the `spwn` binary, as `spwn <command> --help` prints it. ")
	page.WriteString("The page is projected from the Cobra command tree by `make docs` and is never edited by hand; ")
	page.WriteString("the grammar the commands follow is [the CLI](cli.md).\n")
	writeCommand(&page, root)
	return page.Bytes()
}

func writeCommand(page *bytes.Buffer, cmd *cobra.Command) {
	fmt.Fprintf(page, "\n## %s\n\n", cmd.CommandPath())
	page.WriteString(sentence(cmd.Short))
	page.WriteString("\n")

	if long := strings.TrimRight(cmd.Long, " \n"); strings.TrimSpace(long) != "" && strings.TrimSpace(long) != strings.TrimSpace(cmd.Short) {
		for _, paragraph := range strings.Split(strings.Trim(long, "\n"), "\n\n") {
			fmt.Fprintf(page, "\n```text\n%s\n```\n", strings.Trim(paragraph, "\n"))
		}
	}

	if cmd.Runnable() {
		fmt.Fprintf(page, "\n```text\n%s\n```\n", cmd.UseLine())
	}

	if flags := strings.TrimRight(cmd.NonInheritedFlags().FlagUsages(), "\n"); flags != "" {
		fmt.Fprintf(page, "\n```text\n%s\n```\n", flags)
	}

	children := available(cmd)
	if len(children) > 0 {
		page.WriteString("\n")
		for _, child := range children {
			fmt.Fprintf(page, "- [%s](#%s) — %s\n", child.CommandPath(), anchor(child.CommandPath()), sentence(child.Short))
		}
	}

	for _, child := range children {
		writeCommand(page, child)
	}
}

func available(cmd *cobra.Command) []*cobra.Command {
	var children []*cobra.Command
	for _, child := range cmd.Commands() {
		if child.IsAvailableCommand() && !child.IsAdditionalHelpTopicCommand() {
			children = append(children, child)
		}
	}
	return children
}

func anchor(heading string) string {
	return strings.ReplaceAll(strings.ToLower(heading), " ", "-")
}

func sentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasSuffix(text, ".") {
		return text
	}
	return text + "."
}
