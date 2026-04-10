package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/imagebuilder/probe"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your spwn environment for issues",
	Long: `Run diagnostic checks on your spwn environment.

Verifies Docker connectivity, images, configuration files, agents,
and authentication. Reports issues with suggested fixes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New(quiet, verbose, jsonOutput)
		s.Blank()

		passed := 0
		total := 0

		// 1. Docker running
		total++
		if detail, ok := checkDocker(); ok {
			s.Done("Docker", detail)
			passed++
		} else {
			s.Fail("Docker", fmt.Errorf("%s", detail))
		}

		// 2. Base image
		total++
		if detail, ok := checkImage("spwn/world:latest"); ok {
			s.Done("Base image", detail)
			passed++
		} else {
			s.Fail("Base image", fmt.Errorf("%s", detail))
		}

		// 3. Test image (optional)
		total++
		if detail, ok := checkImage("spwn-test:latest"); ok {
			s.Done("Test image", detail)
			passed++
		} else {
			s.Warn("Test image", detail)
			// Count as passed since it's optional
		}

		// 4. Config directory
		total++
		baseDir := foundation.BaseDir()
		if info, err := os.Stat(baseDir); err == nil && info.IsDir() {
			s.Done("Config", abbreviatePath(baseDir)+" exists")
			passed++
		} else {
			s.Fail("Config", fmt.Errorf("%s not found", abbreviatePath(baseDir)))
		}

		// 5. Universe manifest
		total++
		if org, err := universe.LoadOrg(); err == nil && org != nil {
			name := org.Name
			if name == "" {
				name = "unnamed"
			}
			s.Done("Universe", fmt.Sprintf("org.yaml (%s)", name))
			passed++
		} else {
			s.Fail("Universe", fmt.Errorf("org.yaml not found"))
		}

		// 6. World configs
		total++
		if configs, ok := countWorldConfigs(); ok {
			s.Done("World configs", configs)
			passed++
		} else {
			s.Fail("World configs", fmt.Errorf("%s", configs))
		}

		// 7. Agents
		total++
		if agents, err := agentDomain.ListAgents(); err == nil && len(agents) > 0 {
			names := make([]string, 0, len(agents))
			for _, a := range agents {
				names = append(names, a.Name)
			}
			detail := fmt.Sprintf("%d agents", len(agents))
			if len(names) <= 5 {
				detail += " (" + strings.Join(names, ", ") + ")"
			}
			s.Done("Agents", detail)
			passed++
		} else {
			s.Fail("Agents", fmt.Errorf("no agents found"))
		}

		// 8. Auth token
		total++
		if detail, ok := checkAuth(); ok {
			s.Done("Auth", detail)
			passed++
		} else {
			s.Warn("Auth", detail)
		}

		// 9. Version
		total++
		s.Done("Version", "v"+Version)
		passed++

		// Summary
		s.Blank()
		summary := fmt.Sprintf("%d/%d checks passed.", passed, total)
		if passed == total {
			s.Success(summary)
		} else {
			fmt.Fprintf(os.Stderr, "  %s\n", summary)
			printSuggestions(s, passed, total)
		}
		s.Blank()

		return nil // Always exit 0
	},
}

// checkDocker verifies Docker is running and returns version info. Shares
// the same probe used by the observatory API so the CLI and the desktop
// app always agree on whether Docker is healthy.
func checkDocker() (string, bool) {
	st := probe.CheckDocker(context.Background())
	if !st.OK() {
		msg := st.Summary()
		if st.Hint != "" {
			msg += " — " + st.Hint
		}
		return msg, false
	}
	return st.Summary(), true
}

// checkImage verifies a Docker image exists and returns its size.
func checkImage(image string) (string, bool) {
	if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
		return fmt.Sprintf("%s not found", image), false
	}

	out, err := exec.Command("docker", "image", "inspect", "--format", "{{.Size}}", image).Output()
	if err != nil {
		return image, true
	}

	sizeStr := strings.TrimSpace(string(out))
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return fmt.Sprintf("%s (%s)", image, formatBytes(size)), true
	}

	return image, true
}

// countWorldConfigs counts YAML files in ~/.spwn/worlds/.
func countWorldConfigs() (string, bool) {
	worldsDir := foundation.WorldsDir()
	entries, err := os.ReadDir(worldsDir)
	if err != nil {
		return "no world configs found", false
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".yaml"), ".yml")
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return "no world configs found", false
	}

	detail := fmt.Sprintf("%d configs", len(names))
	if len(names) <= 5 {
		detail += " (" + strings.Join(names, ", ") + ")"
	}
	return detail, true
}

// checkAuth verifies authentication is configured.
func checkAuth() (string, bool) {
	baseDir := foundation.BaseDir()
	tokenPath := filepath.Join(baseDir, ".auth-token")
	if data, err := os.ReadFile(tokenPath); err == nil && strings.TrimSpace(string(data)) != "" {
		return "subscription (cached token)", true
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "API key (ANTHROPIC_API_KEY)", true
	}
	return "not configured — run 'claude setup-token' or set ANTHROPIC_API_KEY", false
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b int64) string {
	const (
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%dMB", b/mb)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// printSuggestions prints actionable fix suggestions based on failed checks.
func printSuggestions(s *ui.Stepper, passed, total int) {
	// The specific failures are already shown inline via Fail/Warn.
	// Just add a general hint.
	if passed < total {
		fmt.Fprintf(os.Stderr, "  Run 'spwn init' for first-time setup, or 'make build-base-image' for images.\n")
	}
}
