package universe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jterrazz/spwn/cli/ui"
	"github.com/jterrazz/spwn/domains/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active universes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		j, _ := cmd.Flags().GetBool("json")
		q, _ := cmd.Flags().GetBool("quiet")

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		universes, err := arc.List(ctx)
		if err != nil {
			return fmt.Errorf("error: cannot list universes.\n%w", err)
		}

		if j {
			data, _ := json.MarshalIndent(map[string]interface{}{"active": universes}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if q {
			return nil
		}

		if len(universes) == 0 {
			s.Blank()
			s.Success("No active universes.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "ID", "AGENT", "STATUS", "CREATED")
		for _, u := range universes {
			agent := "—"
			if u.AgentID != "" {
				agent = u.AgentID
			}
			t.AddRow(u.ID, agent, string(u.Status), timeAgo(u.CreatedAt))
		}
		t.Render()

		return nil
	},
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
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
