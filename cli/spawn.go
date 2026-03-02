package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jterrazz/universe/cli/ui"
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
	spawnGate      []string
)

func init() {
	spawnCmd.Flags().StringVarP(&spawnAgent, "agent", "a", "default", "Agent name")
	spawnCmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	spawnCmd.Flags().StringVarP(&spawnUniverse, "universe", "u", "", "Explicit path to a YAML config file")
	spawnCmd.Flags().BoolVarP(&spawnDetach, "detach", "d", false, "Run in background")
	spawnCmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	spawnCmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge element from Substrate: "source:as:cap1,cap2"`)

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
		s := ui.New(quiet, verbose, jsonOutput)

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

		// Parse --gate flags and merge with manifest gates
		for _, g := range spawnGate {
			bridge, err := parseGateFlag(g)
			if err != nil {
				return fmt.Errorf("error: invalid --gate flag %q.\n%w", g, err)
			}
			m.Gate = append(m.Gate, bridge)
		}

		// Build spawn opts
		agentName := ""
		if !spawnNoAgent {
			agentName = spawnAgent
		}

		arc, err := architect.NewFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Start("Spawning universe...")

		result, err := arc.Spawn(ctx, architect.SpawnOpts{
			ConfigName: configName,
			AgentName:  agentName,
			Workspace:  spawnWorkspace,
			Manifest:   m,
			LogWriter:  s.Writer(),
			OnProgress: func(event, detail string) {
				switch event {
				case "image_ready":
					s.Done("Built image", detail)
					s.Start("Spawning universe...")
				case "container_created":
					s.Done("Spawned universe", detail)
				case "mind_mounted":
					s.Done("Mounted mind", detail)
				case "gates_bridged":
					s.Done("Bridged gate", detail)
				case "faculties_generated":
					s.Done("Generated faculties", detail)
				}
			},
		})
		if err != nil {
			s.Fail("Spawn failed", err)
			return err
		}

		u := result.Universe

		// Show non-fatal warnings
		for _, w := range result.Warnings {
			s.Warn("Warning", w)
		}

		// Spawn agent if not --no-agent
		if agentName != "" {
			if spawnDetach {
				s.Start("Spawning agent...")
				if err := arc.SpawnAgentDetached(ctx, u.ID, agentName); err != nil {
					s.Fail("Agent spawn failed", err)
					return err
				}
				s.Done("Agent spawned", "detached")
			} else {
				s.Blank()
				s.Success("Agent is alive.")
				s.Blank()
				if err := arc.SpawnAgent(ctx, u.ID, agentName); err != nil {
					return err
				}
			}
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
		} else if spawnNoAgent || spawnDetach {
			s.Blank()
			if spawnNoAgent {
				s.Success("Universe spawned.")
			} else {
				s.Success("Agent is alive.")
			}
			s.Blank()
			s.Info("Universe:", u.ID)
			if u.AgentID != "" {
				s.Info("Agent:", u.AgentID)
			}
			s.Info("Status:", string(u.Status))
		}

		return nil
	},
}

// parseGateFlag parses "source:as:cap1,cap2" into a GateBridge.
func parseGateFlag(s string) (config.GateBridge, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return config.GateBridge{}, fmt.Errorf("expected format \"source:as[:cap1,cap2]\", got %q", s)
	}

	bridge := config.GateBridge{
		Source: parts[0],
		As:     parts[1],
	}

	if len(parts) == 3 && parts[2] != "" {
		bridge.Capabilities = strings.Split(parts[2], ",")
	}

	return bridge, nil
}
