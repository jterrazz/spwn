package universe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jterrazz/spwn/cli/ui"
	"github.com/jterrazz/spwn/domains/gate"
	"github.com/jterrazz/spwn/domains/universe"
	"github.com/spf13/cobra"
)

var (
	spawnConfig    string
	spawnAgent     string
	spawnWorkspace string
	spawnUniverse  string
	spawnDetach    bool
	spawnNoAgent   bool
	spawnGate      []string
)

func init() {
	Cmd.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named universe config (default: default)")
	Cmd.Flags().StringVarP(&spawnAgent, "agent", "a", "default", "Agent name")
	Cmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	Cmd.Flags().StringVarP(&spawnUniverse, "universe", "u", "", "Explicit path to a YAML config file")
	Cmd.Flags().BoolVarP(&spawnDetach, "detach", "d", false, "Run in background")
	Cmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	Cmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge element from Host: "source:as:cap1,cap2"`)
}

// Cmd is the universe command — spawns a universe when run directly,
// and groups subcommands (list, inspect, logs, attach, destroy).
var Cmd = &cobra.Command{
	Use:   "universe",
	Short: "Spawn a universe — an isolated reality for agents",
	Long: `Spawn a universe — the Big Bang.

Creates a world and brings an agent to life inside it. Uses a named universe
config from ~/.spwn/universes/ (default: default.yaml). Specify a config with
the -c flag.

Subcommands: list, inspect, logs, attach, destroy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		// Resolve config name
		configName := "default"
		if spawnConfig != "" {
			configName = spawnConfig
		}

		// Load manifest
		var (
			m   universe.UniverseManifest
			err error
		)
		if spawnUniverse != "" {
			m, err = universe.LoadManifestPath(spawnUniverse)
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

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Start("Spawning universe...")

		result, err := arc.Spawn(ctx, universe.SpawnOpts{
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

		j, _ := cmd.Flags().GetBool("json")
		if j {
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

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

// parseGateFlag parses "source:as:cap1,cap2" into a GateBridge.
func parseGateFlag(s string) (gate.GateBridge, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return gate.GateBridge{}, fmt.Errorf("expected format \"source:as[:cap1,cap2]\", got %q", s)
	}

	bridge := gate.GateBridge{
		Source: parts[0],
		As:     parts[1],
	}

	if len(parts) == 3 && parts[2] != "" {
		bridge.Capabilities = strings.Split(parts[2], ",")
	}

	return bridge, nil
}
