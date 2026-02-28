package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/architect"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all universes",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := architect.New()
			if err != nil {
				return err
			}

			universes, err := a.List(cmd.Context())
			if err != nil {
				return err
			}

			if len(universes) == 0 {
				fmt.Println("No universes found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tMIND\tIMAGE\tSTATUS")

			for _, u := range universes {
				mind := "-"
				if u.Mind != "" {
					mind = u.Mind
				}
				shortID := u.ID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", shortID, mind, u.Image, u.Status)
			}

			return w.Flush()
		},
	}
}
