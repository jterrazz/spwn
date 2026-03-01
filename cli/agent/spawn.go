package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/state"
)

var (
	spawnAgentName   string
	spawnImportFile  string
)

var spawnAgentCmd = &cobra.Command{
	Use:   "spawn <universe-id>",
	Short: "Bring an agent to life inside a universe",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpawnAgent,
}

func init() {
	spawnAgentCmd.Flags().StringVarP(&spawnAgentName, "agent", "a", "", "Agent name (required)")
	spawnAgentCmd.Flags().StringVar(&spawnImportFile, "import", "", "Import Mind from tar.gz before spawning")
	spawnAgentCmd.MarkFlagRequired("agent")
}

func runSpawnAgent(cmd *cobra.Command, args []string) error {
	universeID := args[0]

	docker, err := backend.NewDocker()
	if err != nil {
		return fmt.Errorf("error: cannot connect to Docker.\nIs Docker running? Try: docker info")
	}

	store, err := state.NewStore()
	if err != nil {
		return err
	}

	arch := architect.New(docker, store)

	// Import Mind from archive if specified.
	if spawnImportFile != "" {
		fmt.Println()
		fmt.Printf("  Importing Mind from %s...\n", spawnImportFile)
		if err := mind.Import(spawnAgentName, spawnImportFile); err != nil {
			return fmt.Errorf("error: failed to import Mind.\n%s", err)
		}
		fmt.Println("  ✓ Mind imported")
	}

	fmt.Println()
	fmt.Printf("  Spawning agent into universe %s...\n", universeID)
	fmt.Println()

	exitCode, err := arch.SpawnAgent(cmd.Context(), universeID, spawnAgentName)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Agent exited (code %d). Universe is idle.\n", exitCode)
	fmt.Println()

	return nil
}
