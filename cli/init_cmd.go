package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate universe.yaml in the current directory",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	const filename = "universe.yaml"

	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("error: %s already exists in the current directory", filename)
	}

	content := fmt.Sprintf(`# universe.yaml — declares the physics of a reality

physics:
  origin: %s

  constants:
    cpu: %d
    memory: %s
    disk: %s
    timeout: %s

  laws:
    network: %s
    max-processes: %d

  elements: []

# gate: []
`, config.DefaultOrigin,
		config.DefaultCPU,
		config.DefaultMemory,
		config.DefaultDisk,
		config.DefaultTimeout,
		config.DefaultNetwork,
		config.DefaultMaxProcs,
	)

	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("  ✓ Created %s (physics.origin: %s)\n", filename, config.DefaultOrigin)
		fmt.Println()
		fmt.Println("  Next: universe spawn --agent <name>")
		fmt.Println()
	}

	return nil
}
