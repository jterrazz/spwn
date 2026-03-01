package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/spf13/cobra"
)

var (
	spawnAgent     string
	spawnWorkspace string
	spawnUniverse  string
	spawnDetach    bool
	spawnNoAgent   bool
)

func init() {
	spawnCmd.Flags().StringVarP(&spawnAgent, "agent", "a", "default", "Agent name")
	spawnCmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	spawnCmd.Flags().StringVarP(&spawnUniverse, "universe", "u", "", "Explicit path to a YAML config file")
	spawnCmd.Flags().BoolVarP(&spawnDetach, "detach", "d", false, "Run in background")
	spawnCmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")

	rootCmd.AddCommand(spawnCmd)
}

var spawnCmd = &cobra.Command{
	Use:   "spawn [name]",
	Short: "Create a universe and spawn an agent inside it",
	Long: `Create a universe — the Big Bang.

Spawns a world and brings an agent to life inside it. Uses a named universe
config from ~/.universe/universes/ (default: default.yaml). The config name
is an optional positional argument.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve config name
		configName := "default"
		if len(args) > 0 {
			configName = args[0]
		}

		// Load manifest
		var (
			m   config.UniverseManifest
			err error
		)
		if spawnUniverse != "" {
			m, err = manifest.LoadPath(spawnUniverse)
		} else {
			m, err = manifest.Load(configName)
		}
		if err != nil {
			return fmt.Errorf("error: cannot load config %q.\n%w", configName, err)
		}

		if err := manifest.Validate(m); err != nil {
			return fmt.Errorf("error: invalid config.\n%w", err)
		}

		// Build spawn opts
		agentName := ""
		if !spawnNoAgent {
			agentName = spawnAgent
		}

		arc, err := newArchitect()
		if err != nil {
			return err
		}

		if !quiet {
			fmt.Println("\n  Spawning universe...")
			fmt.Println()
		}

		u, err := arc.Spawn(ctx, architect.SpawnOpts{
			ConfigName: configName,
			AgentName:  agentName,
			Workspace:  spawnWorkspace,
			Manifest:   m,
		})
		if err != nil {
			return fmt.Errorf("error: spawn failed.\n%w", err)
		}

		if !quiet {
			fmt.Printf("  ✓ Provisioned container (ubuntu:24.04)\n")
			if spawnWorkspace != "" {
				fmt.Printf("  ✓ Mounted workspace %s → /workspace\n", spawnWorkspace)
			}
			fmt.Printf("  ✓ Generated faculties.md\n")
			if agentName != "" {
				fmt.Printf("  ✓ Mounted Mind %s → /mind\n", agentName)
			}
		}

		// Spawn agent if not --no-agent
		if agentName != "" {
			if spawnDetach {
				if err := arc.SpawnAgentDetached(ctx, u.ID, agentName); err != nil {
					return fmt.Errorf("error: agent spawn failed.\n%w", err)
				}
				if !quiet {
					fmt.Printf("  ✓ Spawned Claude Code CLI (detached)\n")
				}
			} else {
				if !quiet {
					fmt.Printf("  ✓ Spawning Claude Code CLI...\n\n")
				}
				if err := arc.SpawnAgent(ctx, u.ID, agentName); err != nil {
					return fmt.Errorf("error: agent spawn failed.\n%w", err)
				}
			}
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
		} else if !quiet && (spawnNoAgent || spawnDetach) {
			fmt.Println()
			if spawnNoAgent {
				fmt.Println("  Universe spawned.")
			} else {
				fmt.Println("  Universe spawned. Agent is alive.")
			}
			fmt.Println()
			fmt.Printf("  Universe:  %s\n", u.ID)
			if u.AgentID != "" {
				fmt.Printf("  Agent:     %s\n", u.AgentID)
			}
			fmt.Printf("  Status:    %s\n", u.Status)
		}

		return nil
	},
}
