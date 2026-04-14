// Package profile implements the `spwn profile` command group - management
// of reusable personality profile templates. Profiles are first-class
// composable blocks alongside tools and skills: stackable, shareable,
// and attached to agents via "spwn agent add <name> --profile <name>".
//
// A profile template is a markdown file declaring role, tone, purpose,
// and behavior. It replaces the legacy "persona" concept.
package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/base"
	"github.com/spf13/cobra"
)

// Cmd is the root `spwn profile` command group.
var Cmd = &cobra.Command{
	Use:   "profile",
	Short: "Author and manage reusable profile templates (personality)",
	Long: `Profiles are reusable personality templates - role, tone, purpose, behavior.

A profile defines WHO the agent is. Tools and skills define what it can do.

Attach one to an agent with:
  spwn agent add <agent> --profile <profile-name>`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(newCmd)
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(rmCmd)

	Cmd.SetHelpFunc(profileHelp)
}

func profileHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "profile" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ profile")+" "+ui.Faint("- reusable personality templates"),
		[]ui.HelpGroup{
			{Title: "Author", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List profile templates"},
				{Name: "new <name>", Desc: "Author a new profile"},
				{Name: "edit <name>", Desc: "Open a profile in $EDITOR"},
				{Name: "show <name>", Desc: "Display a profile"},
				{Name: "rm <name>", Desc: "Delete a profile"},
			}},
			{Title: "Registry", Commands: []ui.HelpEntry{
				{Name: "get <ref>", Desc: "Install a shared profile " + ui.Faint("[planned]")},
				{Name: "publish <name>", Desc: "Publish a profile " + ui.Faint("[planned]")},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn profile new researcher", Desc: ""},
				{Name: "spwn agent add neo --profile researcher", Desc: ""},
			}},
		},
		"spwn profile [command]",
		"",
	)
}

// profilesDir returns the root directory for user profile templates.
func profilesDir() string {
	return filepath.Join(base.BaseDir(), "profiles")
}

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List profile templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := profilesDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(cmd.OutOrStderr(), "No profiles authored yet.")
				fmt.Fprintln(cmd.OutOrStderr(), "Create one with 'spwn profile new <name>'.")
				return nil
			}
			return fmt.Errorf("read %s: %w", dir, err)
		}
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStderr(), "No profiles authored yet.")
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
	Use:   "new <profile-name>",
	Short: "Author a new profile template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := profilesDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		path := filepath.Join(dir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("profile %q already exists at %s", name, path)
		}
		template := fmt.Sprintf(`---
name: %s
role: specialist
---

# %s

> One-line summary of this personality.

You are %s. Describe the agent's role, expertise, and purpose here.

## Style

How does this agent communicate? What's its tone?

## Values

What does this agent stand for? What won't it compromise on?

## Behavior

How does this agent approach problems? What's its default posture?
`, name, name, name)
		if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Authored %s\n", path)
		fmt.Fprintln(cmd.OutOrStderr(), "Edit with 'spwn profile edit "+name+"' or open the file directly.")
		fmt.Fprintln(cmd.OutOrStderr(), "Attach to an agent: spwn agent add <agent> --profile "+name)
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <profile-name>",
	Short: "Open a profile template in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(profilesDir(), name+".md")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("profile %q not found at %s", name, path)
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
	Use:   "show <profile-name>",
	Short: "Display a profile template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(profilesDir(), name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("profile %q not found at %s", name, path)
		}
		fmt.Fprintln(cmd.OutOrStderr(), string(data))
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <profile-ref>",
	Short: "Install a profile template from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "install %q: the profile registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish <profile-name>",
	Short: "Publish a profile template to the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStderr(), "publish %q: the profile registry is not yet available.\n", args[0])
		fmt.Fprintln(cmd.OutOrStderr(), "The registry is planned for a future release.")
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:     "rm <profile-name>",
	Aliases: []string{"remove", "delete"},
	Short:   "Delete a profile template",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join(profilesDir(), name+".md")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("profile %q not found at %s", name, path)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Removed %s\n", path)
		return nil
	},
}
