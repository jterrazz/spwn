package snap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"

	"github.com/spf13/cobra"
)

var (
	snapName      string
	snapConfig    string
	snapAgent     string
	snapWorkspace string
)

func init() {
	saveCmd.Flags().StringVar(&snapName, "name", "", "Name for the snapshot")

	restoreCmd.Flags().StringVarP(&snapConfig, "config", "c", "", "Named world config (default: default)")
	restoreCmd.Flags().StringVarP(&snapAgent, "agent", "a", "default", "Agent name")
	restoreCmd.Flags().StringVarP(&snapWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")

	Cmd.AddCommand(saveCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(restoreCmd)
	Cmd.AddCommand(rmCmd)
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

func dockerHint(err error) error {
	if strings.Contains(err.Error(), "cannot connect to Docker") {
		return fmt.Errorf("Docker is not running")
	}
	return err
}

// --- save ---

var saveCmd = &cobra.Command{
	Use:   "save <world-id>",
	Short: "Save world state as a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		s.Start("Saving snapshot...")
		tag, err := arc.Snapshot(context.Background(), worldID, snapName)
		if err != nil {
			return s.FailHint("Snapshot failed", err, "Check that the world is running with \"spwn ls\"")
		}

		s.Done("Saved snapshot", tag)
		return nil
	},
}

// --- ls ---

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all snapshots",
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

// --- restore ---

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

		imageTag := "spwn-snapshot:" + snapshotRef

		configName := snapConfig
		if configName == "" {
			configName = "default"
		}
		m, err := universe.LoadManifest(configName)
		if err != nil {
			return fmt.Errorf("cannot load config %q: %w", configName, err)
		}
		universe.ApplyDefaults(&m)

		opts := universe.SpawnOpts{
			ConfigName: configName,
			AgentName:  snapAgent,
			Workspace:  snapWorkspace,
			Manifest:   m,
			Image:      imageTag,
		}

		s.Start("Restoring from snapshot...")
		result, err := arc.Spawn(context.Background(), opts)
		if err != nil {
			return s.FailHint("Restore failed", err, "Check available snapshots with \"spwn snap ls\"")
		}

		s.Done("Restored world", result.Universe.ID)
		s.Info("From snapshot:", snapshotRef)
		return nil
	},
}

// --- rm ---

var rmCmd = &cobra.Command{
	Use:   "rm <snapshot>",
	Short: "Remove a snapshot",
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
			return s.FailHint("Delete failed", err, "Check available snapshots with \"spwn snap ls\"")
		}

		s.Done("Deleted snapshot", snapshotRef)
		return nil
	},
}

// --- helpers ---

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

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "\u2014"
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
