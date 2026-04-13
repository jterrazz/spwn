package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agentDomain "spwn.sh/packages/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:   "import <path-to-tar.gz>",
	Short: "Import an agent from a tar.gz archive",
	Long: `Import an agent's Mind from a previously exported tar.gz archive.

The agent name is derived from the archive filename (e.g., neo.tar.gz → neo).
The archive must contain at least an identity/ layer.`,
	Example: `  spwn agent import neo.tar.gz
  spwn agent import /path/to/backup.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		archivePath := args[0]
		s := newStepper(cmd)

		// Validate the archive exists
		if _, err := os.Stat(archivePath); err != nil {
			s.Blank()
			return s.FailHint("Archive not found", err,
				"Provide the path to a .tar.gz file created by \"spwn agent export\"")
		}

		// Derive agent name from filename: neo.tar.gz → neo
		base := filepath.Base(archivePath)
		name := strings.TrimSuffix(base, ".tar.gz")
		if name == base {
			// Try .tgz as well
			name = strings.TrimSuffix(base, ".tgz")
		}
		if name == "" || name == base {
			s.Blank()
			return s.FailHint("Invalid archive", fmt.Errorf("cannot derive agent name from %q", base),
				"Archive should be named <agent-name>.tar.gz")
		}

		// Check if agent already exists
		agentDir := agentDomain.AgentDir(name)
		if _, err := os.Stat(agentDir); err == nil {
			s.Blank()
			return s.FailHint("Agent already exists", fmt.Errorf("agent %q already exists", name),
				fmt.Sprintf("Remove it first with \"spwn agent rm %s\"", name))
		}

		s.Blank()
		s.Start(fmt.Sprintf("Importing agent %s...", name))

		if err := agentDomain.ImportMind(name, archivePath); err != nil {
			// Clean up partial import
			os.RemoveAll(agentDir)
			return s.FailHint("Import failed", err,
				"Check that the archive is a valid tar.gz created by \"spwn agent export\"")
		}

		// Validate extracted structure has core/ layer (or legacy identity/)
		coreDir := filepath.Join(agentDir, "core")
		identityDir := filepath.Join(agentDir, "identity")
		if _, err := os.Stat(coreDir); err != nil {
			if _, errI := os.Stat(identityDir); errI != nil {
				os.RemoveAll(agentDir)
				s.Blank()
				return s.FailHint("Invalid archive content", fmt.Errorf("missing core/ layer"),
					"Archive must contain at least a core/ directory")
			}
		}

		s.Done("Imported agent", name)
		s.Blank()

		return nil
	},
}
