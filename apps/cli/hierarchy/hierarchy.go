package hierarchy

import (
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

// Cmd is the top-level `spwn hierarchy` command.
var Cmd = &cobra.Command{
	Use:   "hierarchy",
	Short: "Manage hierarchies — list and inspect role structures",
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(inspectCmd)
}

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New(false, false, false)
}

// ── spwn hierarchy ls ──

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all hierarchies",
	RunE: func(cmd *cobra.Command, args []string) error {
		hierarchies, err := agentDomain.ListHierarchies()
		if err != nil {
			return fmt.Errorf("cannot list hierarchies: %w", err)
		}

		if len(hierarchies) == 0 {
			s := newStepper(cmd)
			s.Blank()
			s.Info("Hierarchies:", "None found.")
			s.Log("Create one with: spwn hierarchy new")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "SLUG", "NAME", "ROLES", "DESCRIPTION")
		for _, h := range hierarchies {
			roleNames := make([]string, 0, len(h.Roles))
			for _, r := range h.Roles {
				roleNames = append(roleNames, r.Name)
			}
			t.AddRow(
				h.Slug,
				h.Name,
				strings.Join(roleNames, ", "),
				h.Description,
			)
		}
		t.Render()
		return nil
	},
}

// ── spwn hierarchy inspect ──

var inspectCmd = &cobra.Command{
	Use:   "inspect <slug>",
	Short: "Show roles in a hierarchy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		h, err := agentDomain.GetHierarchy(slug)
		if err != nil {
			return fmt.Errorf("cannot load hierarchy: %w", err)
		}

		w := cmd.ErrOrStderr()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s  %s\n", ui.Strong(h.Name), ui.Faint(h.Slug))
		if h.Description != "" {
			fmt.Fprintf(w, "  %s\n", ui.Faint(h.Description))
		}
		fmt.Fprintln(w)

		t := ui.NewTable(ui.ModeNormal, "ROLE", "LEVEL", "REPORTS TO", "CAN COMMAND", "PERMISSIONS")
		for _, r := range h.Roles {
			reportsTo := r.ReportsTo
			if reportsTo == "" {
				reportsTo = "\u2014"
			}
			canCommand := "\u2014"
			if len(r.CanCommand) > 0 {
				canCommand = strings.Join(r.CanCommand, ", ")
			}
			permissions := "\u2014"
			if len(r.Permissions) > 0 {
				permissions = strings.Join(r.Permissions, ", ")
			}
			t.AddRow(
				r.Name,
				fmt.Sprintf("%d", r.Level),
				reportsTo,
				canCommand,
				permissions,
			)
		}
		t.Render()
		fmt.Fprintln(w)
		return nil
	},
}
