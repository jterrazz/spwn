package world

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/gate"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	spawnConfig    string
	spawnAgent     string
	spawnWorkspace string
	spawnWorld     string
	spawnDetach    bool
	spawnNoAgent   bool
	spawnGate      []string
	spawnGovernor  string
)

func init() {
	Cmd.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	Cmd.Flags().StringVarP(&spawnAgent, "agent", "a", "default", "Agent name")
	Cmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	Cmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	Cmd.Flags().BoolVarP(&spawnDetach, "detach", "d", false, "Run in background")
	Cmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	Cmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge element from Host: "source:as:cap1,cap2"`)
	Cmd.Flags().StringVar(&spawnGovernor, "governor", "", "Governor agent for this world")
}

// Cmd is the world command — spawns a world when run directly,
// and groups subcommands (list, inspect, logs, attach, destroy).
var Cmd = &cobra.Command{
	Use:   "world",
	Short: "Spawn a world — an isolated reality for agents",
	Long: `Spawn a world — the Big Bang.

Creates an isolated Docker environment and brings an agent to life inside it.
Uses a named world config from ~/.spwn/worlds/ (default: default.yaml).`,
	Example: `  spwn world -w .                    Spawn with current directory
  spwn world -c acme -w ~/project   Named config + workspace
  spwn world --governor morpheus     With a governor agent
  spwn world --no-agent              Empty world (no agent)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no flags set at all, show help instead of spawning with defaults
		if !cmd.Flags().Changed("config") && !cmd.Flags().Changed("agent") &&
			!cmd.Flags().Changed("workspace") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("detach") && !cmd.Flags().Changed("no-agent") &&
			!cmd.Flags().Changed("governor") && !cmd.Flags().Changed("gate") {
			return cmd.Help()
		}

		ctx := context.Background()
		s := newStepper(cmd)

		// Resolve config name
		configName := "default"
		if spawnConfig != "" {
			configName = spawnConfig
		}

		// Load manifest
		var (
			m   universe.Manifest
			err error
		)
		if spawnWorld != "" {
			m, err = universe.LoadManifestPath(spawnWorld)
		} else {
			m, err = universe.LoadManifest(configName)
		}
		if err != nil {
			return fmt.Errorf("error: cannot load config %q.\n%w", configName, err)
		}

		if err := universe.ValidateManifest(m); err != nil {
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

		// Build multi-agent list when --governor is used
		var agents []universe.AgentSpec
		if spawnGovernor != "" {
			agents = append(agents, universe.AgentSpec{Name: spawnGovernor, Tier: "governor"})
			if agentName != "" {
				agents = append(agents, universe.AgentSpec{Name: agentName, Tier: "citizen"})
			}
			// Clear single-agent name since we're using multi-agent
			agentName = ""
		}

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Start("Spawning world...")

		result, err := arc.Spawn(ctx, universe.SpawnOpts{
			ConfigName: configName,
			AgentName:  agentName,
			Workspace:  spawnWorkspace,
			Manifest:   m,
			Agents:     agents,
			LogWriter:  s.Writer(),
			OnProgress: func(event, detail string) {
				switch event {
				case "image_ready":
					s.Done("Built image", detail)
					s.Start("Spawning world...")
				case "container_created":
					s.Done("Spawned world", detail)
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

		// Spawn agents
		if len(agents) > 0 {
			// Multi-agent mode
			s.Start("Spawning colony...")
			if err := arc.SpawnAgents(ctx, u.ID, agents); err != nil {
				s.Fail("Colony spawn failed", err)
				return err
			}
			s.Done("Colony spawned", fmt.Sprintf("%d agent(s)", len(agents)))
		} else if agentName != "" {
			// Single-agent mode (backward compatible)
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

		j, _ := cmd.Flags().GetBool("json")
		if j {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
		} else if spawnNoAgent || spawnDetach {
			s.Blank()
			if spawnNoAgent {
				s.Success("World spawned.")
			} else {
				s.Success("Agent is alive.")
			}
			s.Blank()
			s.Info("World:", u.ID)
			if u.AgentID != "" {
				s.Info("Agent:", u.AgentID)
			}
			s.Info("Status:", string(u.Status))
		}

		return nil
	},
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

// parseGateFlag parses "source:as:cap1,cap2" into a GateBridge.
func parseGateFlag(s string) (gate.Bridge, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return gate.Bridge{}, fmt.Errorf("expected format \"source:as[:cap1,cap2]\", got %q", s)
	}

	bridge := gate.Bridge{
		Source: parts[0],
		As:     parts[1],
	}

	if len(parts) == 3 && parts[2] != "" {
		bridge.Capabilities = strings.Split(parts[2], ",")
	}

	return bridge, nil
}
