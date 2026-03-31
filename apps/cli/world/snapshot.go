package world

import (
	"context"
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"

	"github.com/spf13/cobra"
)

var snapshotName string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <world-id>",
	Short: "Save a running world as a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		s.Start("Saving snapshot...")
		tag, err := arc.Snapshot(context.Background(), worldID, snapshotName)
		if err != nil {
			s.Fail("Snapshot failed", err)
			return err
		}

		s.Done("Saved snapshot", tag)
		return nil
	},
}

var snapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "List all world snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		snapshots, err := arc.ListSnapshots(context.Background())
		if err != nil {
			return err
		}

		if len(snapshots) == 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "  \u2713 No snapshots.")
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "WORLD", "NAME", "SIZE", "CREATED")
		for _, snap := range snapshots {
			// Parse tag: "spwn-snapshot:w-default-28373--pre-deploy"
			parts := strings.SplitN(strings.TrimPrefix(snap.Tag, "spwn-snapshot:"), "--", 2)
			worldID := parts[0]
			name := ""
			if len(parts) > 1 {
				name = parts[1]
			}
			size := formatSize(snap.Size)
			created := timeAgo(snap.Created)
			t.AddRow(worldID, name, size, created)
		}
		t.Render()
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <snapshot>",
	Short: "Restore a world from a snapshot",
	Long:  `Creates a new world from a previously saved snapshot. The snapshot format is: w-{id}--{name}`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotRef := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		// Build the full image tag
		imageTag := "spwn-snapshot:" + snapshotRef

		// Load manifest
		configName := spawnConfig
		if configName == "" {
			configName = "default"
		}
		m, err := universe.LoadManifest(configName)
		if err != nil {
			return fmt.Errorf("error: cannot load config %q.\n%w", configName, err)
		}
		universe.ApplyDefaults(&m)

		opts := universe.SpawnOpts{
			ConfigName: configName,
			AgentName:  spawnAgent,
			Workspace:  spawnWorkspace,
			Manifest:   m,
			Image:      imageTag,
		}

		s.Start("Restoring from snapshot...")
		result, err := arc.Spawn(context.Background(), opts)
		if err != nil {
			s.Fail("Restore failed", err)
			return err
		}

		s.Done("Restored world", result.Universe.ID)
		s.Info("From snapshot:", snapshotRef)
		return nil
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotRef := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		imageTag := "spwn-snapshot:" + snapshotRef
		s.Start("Deleting snapshot...")
		if err := arc.DeleteSnapshot(context.Background(), imageTag); err != nil {
			s.Fail("Delete failed", err)
			return err
		}

		s.Done("Deleted snapshot", snapshotRef)
		return nil
	},
}

func formatSize(bytes int64) string {
	const (
		MB = 1024 * 1024
		GB = 1024 * 1024 * 1024
	)
	if bytes >= GB {
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	}
	return fmt.Sprintf("%.0fMB", float64(bytes)/float64(MB))
}

func init() {
	snapshotCmd.Flags().StringVar(&snapshotName, "name", "", "Name for the snapshot")

	// snapshot delete is a subcommand of snapshot
	snapshotCmd.AddCommand(snapshotDeleteCmd)

	// Register with parent
	Cmd.AddCommand(snapshotCmd)
	Cmd.AddCommand(snapshotsCmd)
	Cmd.AddCommand(restoreCmd)
}
