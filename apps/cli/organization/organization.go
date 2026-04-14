package organization

import (
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/packages/mind"
	"github.com/spf13/cobra"
)

// Cmd is the top-level `spwn organization` command.
var Cmd = &cobra.Command{
	Use:   "organization",
	Short: "Manage organizations - list and inspect role structures",
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(inspectCmd)
}

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
}

// ── spwn organization ls ──

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		organizations, err := agentDomain.ListOrganizations()
		if err != nil {
			return fmt.Errorf("cannot list organizations: %w", err)
		}

		if len(organizations) == 0 {
			s := newStepper(cmd)
			s.Blank()
			s.Info("Organizations:", "None found.")
			s.Log("Create one with: spwn organization new")
			s.Blank()
			return nil
		}

		t := ui.NewTable("SLUG", "NAME", "ROLES", "DESCRIPTION")
		for _, h := range organizations {
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

// ── spwn organization inspect ──

var inspectCmd = &cobra.Command{
	Use:   "inspect <slug>",
	Short: "Show roles in an organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		h, err := agentDomain.GetOrganization(slug)
		if err != nil {
			return fmt.Errorf("cannot load organization: %w", err)
		}

		w := cmd.ErrOrStderr()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s  %s\n", ui.Strong(h.Name), ui.Faint(h.Slug))
		if h.Description != "" {
			fmt.Fprintf(w, "  %s\n", ui.Faint(h.Description))
		}
		fmt.Fprintln(w)

		t := ui.NewTable("ROLE", "LEVEL", "REPORTS TO", "CAN COMMAND", "PERMISSIONS")
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
