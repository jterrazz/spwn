package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/config"
)

func spawnCmd() *cobra.Command {
	var (
		image     string
		mind      string
		workspace string
		memory    string
		timeout   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Create, start, and spawn an agent in a universe",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := architect.New()
			if err != nil {
				return err
			}

			cfg := &config.UniverseConfig{
				Image:     image,
				Mind:      mind,
				Workspace: workspace,
				Memory:    memory,
				Timeout:   timeout,
			}

			u, err := a.Spawn(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			fmt.Printf("Universe spawned: %s\n", u.ID)
			fmt.Printf("  Image:  %s\n", u.Image)
			if u.Mind != "" {
				fmt.Printf("  Mind:   %s\n", u.Mind)
			}
			fmt.Printf("  Status: %s\n", u.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&image, "image", "ubuntu:24.04", "Docker image to use")
	cmd.Flags().StringVar(&mind, "mind", "", "Mind identity to mount")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Host workspace directory to mount")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit (e.g. 512m, 1g)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout (e.g. 5m, 1h)")

	return cmd
}
