package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jterrazz/universe/internal/manifest"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage universe configs",
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configInspectCmd)
	configCmd.AddCommand(configInitCmd)
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available universe configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := manifest.ListConfigs()
		if err != nil {
			return fmt.Errorf("error: cannot list configs.\n%w", err)
		}

		if len(names) == 0 {
			fmt.Println("\n  No universe configs found.")
			fmt.Println("  Run 'universe init <name>' to get started.")
			return nil
		}

		fmt.Printf("\n  %-18s %-10s %s\n", "NAME", "NETWORK", "TECHNOLOGIES")
		for _, name := range names {
			m, err := manifest.Load(name)
			if err != nil {
				fmt.Printf("  %-18s (error loading)\n", name)
				continue
			}
			network := m.Physics.Laws.Network
			techs := strings.Join(m.Technologies, ", ")
			if techs == "" {
				techs = "(none)"
			}
			fmt.Printf("  %-18s %-10s %s\n", name, network, techs)
		}
		fmt.Println()

		return nil
	},
}

var configInspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show a universe config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		m, err := manifest.Load(name)
		if err != nil {
			return fmt.Errorf("error: universe config %q not found.\nRun 'universe config list' to see available configs.", name)
		}

		fmt.Printf("\n  Universe config: %s\n\n", name)
		fmt.Printf("  Constants:\n")
		fmt.Printf("    CPU: %d | Memory: %s | Disk: %s | Timeout: %s\n",
			m.Physics.Constants.CPU,
			m.Physics.Constants.Memory,
			m.Physics.Constants.Disk,
			m.Physics.Constants.Timeout,
		)
		fmt.Printf("\n  Laws:\n")
		fmt.Printf("    Network: %s\n", m.Physics.Laws.Network)
		fmt.Printf("    Max processes: %d\n", m.Physics.Laws.MaxProcesses)
		fmt.Printf("\n  Technologies:\n")
		if len(m.Technologies) > 0 {
			fmt.Printf("    %s\n", strings.Join(m.Technologies, ", "))
		} else {
			fmt.Printf("    (none)\n")
		}
		if len(m.Gate) > 0 {
			fmt.Printf("\n  Gate Bridges:\n")
			for _, gb := range m.Gate {
				fmt.Printf("    %s → %s [%s]\n", gb.As, gb.Source, strings.Join(gb.Capabilities, ", "))
			}
		}
		fmt.Println()

		_ = m
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a new universe config",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}

		if err := manifest.CreateConfig(name); err != nil {
			return fmt.Errorf("error: cannot create config %q.\n%w", name, err)
		}

		if !quiet {
			home, _ := os.UserHomeDir()
			fmt.Printf("\n  Creating universe config %q...\n\n", name)
			fmt.Printf("  ✓ Created %s/.universe/universes/%s.yaml\n\n", home, name)
			fmt.Printf("  Edit the config, then: universe spawn %s\n\n", name)
		}

		return nil
	},
}
