package agent

import (
	"context"
	"fmt"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/state"
	"github.com/spf13/cobra"
)

var (
	agentSpawnUniverse string
	agentSpawnImport   string
)

func init() {
	spawnCmd.Flags().StringVarP(&agentSpawnUniverse, "universe", "u", "", "Target universe ID")
	spawnCmd.Flags().StringVar(&agentSpawnImport, "import", "", "Import Mind from tar.gz before spawning")
	Cmd.AddCommand(spawnCmd)
}

var spawnCmd = &cobra.Command{
	Use:   "spawn [name]",
	Short: "Bring an agent to life inside an existing universe",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		agentName := "default"
		if len(args) > 0 {
			agentName = args[0]
		}

		// Import Mind archive if requested
		if agentSpawnImport != "" {
			s.Blank()
			s.Start("Importing agent...")
			if err := mind.Import(agentName, agentSpawnImport); err != nil {
				s.Fail("Import failed", err)
				return fmt.Errorf("error: import failed.\n%w", err)
			}
			s.Done("Imported agent", agentName)
		}

		docker, err := backend.NewDocker()
		if err != nil {
			return fmt.Errorf("error: cannot connect to Docker.\n%w", err)
		}

		store, err := state.NewStore()
		if err != nil {
			return fmt.Errorf("error: cannot initialize state store.\n%w", err)
		}

		arc := architect.New(docker, store)

		// Resolve universe ID
		universeID := agentSpawnUniverse
		if universeID == "" {
			universes, err := arc.List(ctx)
			if err != nil {
				return fmt.Errorf("error: cannot list universes.\n%w", err)
			}
			if len(universes) == 0 {
				return fmt.Errorf("error: no active universes.\nRun 'universe spawn --no-agent' first")
			}
			if len(universes) > 1 {
				s.Blank()
				s.Fail("Multiple active universes", fmt.Errorf("specify one with --universe"))
				for _, u := range universes {
					s.Info("", fmt.Sprintf("%-20s (%s)", u.ID, u.Status))
				}
				s.Blank()
				return fmt.Errorf("multiple active universes")
			}
			universeID = universes[0].ID
		}

		s.Blank()
		s.Done("Spawning agent into", universeID)
		s.Blank()

		if err := arc.SpawnAgent(ctx, universeID, agentName); err != nil {
			return fmt.Errorf("error: agent spawn failed.\n%w", err)
		}

		return nil
	},
}
