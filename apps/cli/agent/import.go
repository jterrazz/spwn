package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
)

var importAs string

func init() {
	importCmd.Flags().StringVar(&importAs, "as", "", "Rename the agent on import (instead of using the archive filename)")
	Cmd.AddCommand(importCmd)
	ui.MarkExperimental(importCmd)
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
		s := ui.New()

		// Validate the archive exists
		if _, err := os.Stat(archivePath); err != nil {
			s.Blank()
			return s.FailHint("Archive not found", err,
				"Provide the path to a .tar.gz file created by \"spwn agent export\"")
		}

		// Derive agent name from filename: neo.tar.gz → neo (unless
		// --as was given, which takes precedence).
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
		if importAs != "" {
			name = importAs
		}

		// Check if agent already exists
		agentDir := agent.AgentDir(name)
		if _, err := os.Stat(agentDir); err == nil {
			s.Blank()
			return s.FailHint("Agent already exists", fmt.Errorf("agent %q already exists", name),
				fmt.Sprintf("Remove it first with \"spwn agent rm %s\"", name))
		}

		s.Blank()
		s.Start(fmt.Sprintf("Importing agent %s...", name))

		if err := agent.ImportMind(name, archivePath); err != nil {
			// Clean up partial import
			os.RemoveAll(agentDir)
			return s.FailHint("Import failed", err,
				"Check that the archive is a valid tar.gz created by \"spwn agent export\"")
		}

		// Validate extracted structure has identity/ layer
		identityDir := filepath.Join(agentDir, "identity")
		if _, err := os.Stat(identityDir); err != nil {
			os.RemoveAll(agentDir)
			s.Blank()
			return s.FailHint("Invalid archive content", fmt.Errorf("missing identity/ layer"),
				"Archive must contain at least an identity/ directory")
		}

		// When --as was used, rewrite agent.yaml so the embedded
		// `name:` field matches the new directory name. We keep this
		// best-effort - a missing or unparseable agent.yaml does not
		// fail the import.
		if importAs != "" {
			yamlPath := filepath.Join(agentDir, "agent.yaml")
			if data, err := os.ReadFile(yamlPath); err == nil {
				var m map[string]any
				if err := yaml.Unmarshal(data, &m); err == nil {
					m["name"] = name
					if out, err := yaml.Marshal(m); err == nil {
						_ = os.WriteFile(yamlPath, out, 0o644)
					}
				}
			}
		}

		s.Done("Imported agent", name)
		s.Blank()

		return nil
	},
}
