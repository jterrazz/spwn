package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/session"
)

func mindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mind",
		Short: "Manage persistent agent minds",
	}

	cmd.AddCommand(mindListCmd())
	cmd.AddCommand(mindInspectCmd())
	cmd.AddCommand(mindExportCmd())

	return cmd
}

func mindListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all minds",
		RunE: func(cmd *cobra.Command, args []string) error {
			minds, err := mind.ListMinds()
			if err != nil {
				return err
			}
			if len(minds) == 0 {
				fmt.Println("No minds found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "MIND\tPATH\tVALID\tSESSIONS")

			for _, name := range minds {
				path, err := mind.ResolvePath(name)
				if err != nil {
					continue
				}

				valid := "ok"
				if err := mind.Validate(path); err != nil {
					valid = "INVALID"
				}

				sessions, _ := session.List(name)
				sessionCount := len(sessions)

				fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", name, path, valid, sessionCount)
			}
			w.Flush()
			return nil
		},
	}
}

func mindInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <mind-id>",
		Short: "Inspect a mind's structure and history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mindID := args[0]

			path, err := mind.ResolvePath(mindID)
			if err != nil {
				return err
			}

			fmt.Printf("Mind: %s\n", mindID)
			fmt.Printf("Path: %s\n", path)
			fmt.Println()

			// Subdirectory status.
			fmt.Println("Structure:")
			for _, sub := range mind.Subdirs() {
				dir := filepath.Join(path, sub)
				status := "ok"
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					status = "MISSING"
				}
				fmt.Printf("  %-12s %s\n", sub, status)
			}
			fmt.Println()

			// Sessions.
			sessions, _ := session.List(mindID)
			fmt.Printf("Sessions: %d\n", len(sessions))
			for _, s := range sessions {
				fmt.Printf("  - %s (universe: %s, updated: %s)\n",
					s.SessionID, shortID(s.UniverseID), s.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Println()

			// Recent journal entries.
			entries, _ := journal.List(mindID)
			fmt.Printf("Journal entries: %d\n", len(entries))
			// Show last 5.
			start := 0
			if len(entries) > 5 {
				start = len(entries) - 5
			}
			for _, name := range entries[start:] {
				fmt.Printf("  - %s\n", name)
			}

			return nil
		},
	}
}

func mindExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <mind-id> <destination>",
		Short: "Export a mind as a tar.gz archive",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mindID := args[0]
			dest := args[1]

			path, err := mind.ResolvePath(mindID)
			if err != nil {
				return err
			}

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("mind %q does not exist at %s", mindID, path)
			}

			// Ensure dest has .tar.gz extension.
			if !strings.HasSuffix(dest, ".tar.gz") {
				dest = dest + ".tar.gz"
			}

			tarCmd := exec.Command("tar", "-czf", dest, "-C", filepath.Dir(path), filepath.Base(path))
			tarCmd.Stdout = os.Stdout
			tarCmd.Stderr = os.Stderr
			if err := tarCmd.Run(); err != nil {
				return fmt.Errorf("creating archive: %w", err)
			}

			fmt.Printf("Mind exported: %s → %s\n", mindID, dest)
			return nil
		},
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
