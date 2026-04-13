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
	snapWorkspace []string
)

func init() {
	saveCmd.Flags().StringVar(&snapName, "name", "", "Name for the snapshot")

	restoreCmd.Flags().StringVarP(&snapConfig, "config", "c", "", "Named world config (default: default)")
	restoreCmd.Flags().StringVarP(&snapAgent, "agent", "a", "", "Agent name (omit for an empty world)")
	restoreCmd.Flags().StringArrayVarP(&snapWorkspace, "workspace", "w", nil, `Host dir to mount. Repeatable: "path", "name=path", "name=path:ro". Omit for ephemeral.`)

	Cmd.AddCommand(saveCmd)
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(restoreCmd)
	Cmd.AddCommand(rmCmd)
}

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
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

		t := ui.NewTable("WORLD", "NAME", "SIZE", "CREATED")
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

		workspaces, wsErr := parseSnapWorkspaces(snapWorkspace)
		if wsErr != nil {
			return fmt.Errorf("parse workspace: %w", wsErr)
		}

		opts := universe.SpawnOpts{
			ConfigName: configName,
			AgentName:  snapAgent,
			Workspaces: workspaces,
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

// parseSnapWorkspaces parses -w values into universe.Workspace. Mirrors the
// parser in apps/cli/world so snap restores accept the same syntax.
func parseSnapWorkspaces(flags []string) ([]universe.Workspace, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	result := make([]universe.Workspace, 0, len(flags))
	for i, raw := range flags {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		readOnly := false
		if strings.HasSuffix(raw, ":ro") {
			readOnly = true
			raw = strings.TrimSuffix(raw, ":ro")
		}
		name := ""
		path := raw
		if eq := strings.Index(raw, "="); eq > 0 {
			name = strings.TrimSpace(raw[:eq])
			path = strings.TrimSpace(raw[eq+1:])
		}
		if path == "" {
			return nil, fmt.Errorf("workspace #%d has empty path", i+1)
		}
		if name == "" {
			if i == 0 && len(flags) == 1 {
				name = "default"
			} else {
				name = fmt.Sprintf("w%d", i)
			}
		}
		result = append(result, universe.Workspace{Name: name, Path: path, ReadOnly: readOnly})
	}
	return result, nil
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
