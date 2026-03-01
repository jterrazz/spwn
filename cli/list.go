package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List universes",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	arch, err := newArchitect()
	if err != nil {
		return err
	}

	universes, err := arch.List(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(map[string]any{"active": universes}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(universes) == 0 {
		if !quiet {
			fmt.Println()
			fmt.Println("  No active universes.")
			fmt.Println()
			fmt.Println("  Create one: universe spawn --agent <name> --workspace ./project")
			fmt.Println()
		}
		return nil
	}

	fmt.Println()
	fmt.Println("  ACTIVE")
	fmt.Printf("  %-24s %-16s %-12s %-10s %s\n", "ID", "ORIGIN", "AGENT", "STATUS", "CREATED")
	for _, u := range universes {
		agent := "—"
		if u.Agent != "" {
			agent = u.Agent
		}
		ago := timeAgo(u.CreatedAt)
		fmt.Printf("  %-24s %-16s %-12s %-10s %s\n", u.ID, u.Origin, agent, u.Status, ago)
	}
	fmt.Println()

	return nil
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
