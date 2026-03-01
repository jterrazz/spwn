package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Create a universe — the Big Bang",
	Long: `Create a universe from a manifest or flags.

Reads universe.yaml from the current directory or workspace. Provisions the
backend (Docker container), mounts workspace, and generates physics.md
describing the world's reality.

Pass --agent to also spawn an agent in one step.`,
	RunE: runSpawn,
}

var (
	spawnOrigin    string
	spawnAgent     string
	spawnWorkspace string
	spawnDetach    bool
)

func init() {
	spawnCmd.Flags().StringVarP(&spawnOrigin, "origin", "o", "", "Origin (sets physics.origin, overrides manifest)")
	spawnCmd.Flags().StringVarP(&spawnAgent, "agent", "a", "", "Spawn an agent after creation")
	spawnCmd.Flags().StringVarP(&spawnWorkspace, "workspace", "w", "", "Host directory to mount at /workspace")
	spawnCmd.Flags().BoolVarP(&spawnDetach, "detach", "d", false, "Run in background")

	rootCmd.AddCommand(spawnCmd)
}

func runSpawn(cmd *cobra.Command, args []string) error {
	// Discover manifest.
	var searchDirs []string
	if spawnWorkspace != "" {
		searchDirs = append(searchDirs, spawnWorkspace)
	}
	cwd, _ := os.Getwd()
	searchDirs = append(searchDirs, cwd)

	m, manifestPath, err := manifest.Discover(searchDirs...)
	if err != nil {
		return err
	}

	// Apply CLI flag overrides.
	manifest.MergeFlags(&m, spawnOrigin)

	// Validate.
	if err := manifest.Validate(&m); err != nil {
		return fmt.Errorf("error: invalid manifest.\n%s", err)
	}

	arch, err := newArchitect()
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Println()
		fmt.Println("  Spawning universe...")
		fmt.Println()
	}

	// Spawn universe.
	u, err := arch.Spawn(cmd.Context(), config.SpawnOptions{
		Manifest:  m,
		Workspace: spawnWorkspace,
		AgentName: spawnAgent,
		Detach:    spawnDetach,
	})
	if err != nil {
		return err
	}

	if !quiet {
		fmt.Printf("  ✓ Provisioned container from origin %s\n", u.Origin)
		if spawnWorkspace != "" {
			fmt.Printf("  ✓ Mounted workspace %s → /workspace\n", spawnWorkspace)
		}
		if manifestPath != "" {
			fmt.Printf("  ✓ Loaded manifest from %s\n", manifestPath)
		}
		fmt.Println("  ✓ Generated physics.md")
	}

	// If agent specified and not detached, spawn agent interactively.
	if spawnAgent != "" && !spawnDetach {
		if !quiet {
			fmt.Printf("  ✓ Mounted Mind %q → /mind\n", spawnAgent)
		}

		exitCode, err := arch.SpawnAgent(cmd.Context(), u.ID, spawnAgent)
		if err != nil {
			return err
		}

		if !quiet {
			fmt.Println()
			fmt.Printf("  Agent exited (code %d). Universe is idle.\n", exitCode)
			fmt.Println()
		}

		return nil
	}

	// If agent specified and detached, spawn agent in background.
	if spawnAgent != "" && spawnDetach {
		if !quiet {
			fmt.Printf("  ✓ Mounted Mind %q → /mind\n", spawnAgent)
		}

		if err := arch.SpawnAgentDetached(cmd.Context(), u.ID, spawnAgent); err != nil {
			return err
		}

		if !quiet {
			fmt.Printf("  ✓ Agent %q spawned in background\n", spawnAgent)
		}
	}

	if !quiet {
		fmt.Println()
		if spawnAgent != "" {
			fmt.Println("  Universe spawned. Agent is alive.")
		} else {
			fmt.Println("  Universe spawned.")
		}
		fmt.Println()
		fmt.Printf("  ID:       %s\n", u.ID)
		fmt.Printf("  Origin:   %s\n", u.Origin)
		if spawnAgent != "" {
			fmt.Printf("  Agent:    %s\n", spawnAgent)
		}
		fmt.Printf("  Backend:  %s\n", u.Backend)
		fmt.Printf("  Status:   %s\n", u.Status)
		fmt.Println()

		if spawnAgent == "" {
			fmt.Printf("  Next: universe agent spawn %s --agent <name>\n", u.ID)
			fmt.Println()
		}
	}

	return nil
}
