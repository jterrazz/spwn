// Package skill implements the `spwn skill` command group - authoring and
// managing reusable skill files. Skills are procedures, playbooks, or
// pieces of knowledge authored in markdown that agents can invoke.
//
// Skills are first-class composable blocks alongside tools and profiles.
// Attach one to an agent via "spwn agent add <name> --skill <skill>".
package skill

import (
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
	"spwn.sh/packages/paths"
)

// Cmd is the root `spwn skill` command group.
var Cmd = &cobra.Command{
	Use:   "skill",
	Short: "Author and manage reusable skill files",
	Long: `Skills are procedures, playbooks, or pieces of knowledge - authored in markdown.

Attach one to an agent with:
  spwn agent add <agent> --skill <skill-name>`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(newCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(rmCmd)

	Cmd.SetHelpFunc(skillHelp)

	ui.MarkExperimental(editCmd)
}

func skillHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "skill" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ skill")+" "+ui.Faint("- reusable skill files for agents"),
		[]ui.HelpGroup{
			{Title: "Author", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List skill files"},
				{Name: "new <name>", Desc: "Author a new skill"},
				{Name: "edit <name>", Desc: "Open a skill in $EDITOR"},
				{Name: "show <name>", Desc: "Display a skill"},
				{Name: "rm <name>", Desc: "Delete a skill"},
			}},
			{Title: "Registry", Commands: []ui.HelpEntry{
				{Name: "get <ref>", Desc: "Install a shared skill " + ui.Faint("[planned]")},
				{Name: "publish <name>", Desc: "Publish a skill " + ui.Faint("[planned]")},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn skill new paper-reading", Desc: ""},
				{Name: "spwn agent add neo --skill paper-reading", Desc: ""},
			}},
		},
		"spwn skill [command]",
		"",
	)
}

// skillsDir returns the root directory for user skill files.
func skillsDir() string {
	return filepath.Join(paths.BaseDir(), "skills")
}

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List skill files",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := skillsDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStderr(), "No skills authored yet.")
				fmt.Fprintln(cmd.OutOrStderr(), "Create one with 'spwn skill new <name>'.")
				return nil
			}
			return fmt.Errorf("read %s: %w", dir, err)
		}
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStderr(), "No skills authored yet.")
			return nil
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if filepath.Ext(name) == ".md" {
				fmt.Fprintf(cmd.OutOrStderr(), "  %s\n", name[:len(name)-3])
			}
		}
		return nil
	},
}

var newCmd = &cobra.Command{
	Use:   "new <skill-name>",
	Short: "Author a new skill file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := skillsDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		path := filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("skill %q already exists at %s", name, path)
		}
		template := fmt.Sprintf(`# %s

> One-line description of what this skill does.

## When to use

Describe the trigger. When should the agent invoke this skill?

## Prerequisites

List required tools, environment, or prior knowledge.

## Steps

1. First step
2. Second step
3. Third step

## Rollback

How to undo if things go wrong.
`, name)
		if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Authored %s\n", path)
		fmt.Fprintln(cmd.OutOrStderr(), "Edit with 'spwn skill edit "+name+"' or open the file directly.")
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <skill-name>",
	Short: "Open a skill file in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(skillsDir(), name+".md")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("skill %q not found at %s", name, path)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Open %s in your editor:\n  %s %s\n", name, editor, path)
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <skill-name>",
	Short: "Display a skill file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(skillsDir(), name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("skill %q not found at %s", name, path)
		}
		fmt.Fprintln(cmd.OutOrStderr(), string(data))
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <skill-ref>",
	Short: "Install a skill from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "install %q: the skill registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish <skill-name>",
	Short: "Publish a skill to the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "publish %q: the skill registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:     "rm <skill-name>",
	Aliases: []string{"remove", "delete"},
	Short:   "Delete a skill file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(skillsDir(), name+".md")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("skill %q not found at %s", name, path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Removed %s\n", path)
		return nil
	},
}
