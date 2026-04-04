package dash

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultDashHelp func(*cobra.Command, []string)

// Cmd is the parent command for Dashboard operations.
var Cmd = &cobra.Command{
	Use:   "dash",
	Short: "Visual dashboard",
	Long:  `The dashboard — a real-time visual dashboard showing all worlds, agents, and their evolution.`,
}

var portFlag string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the dashboard (API + web)",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		// Header
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", ui.Strong("⬡ spwn dash"))
		fmt.Fprintln(w)

		// Start the Next.js dashboard in background
		// Find the observatory app — check known paths
		observatoryDir := findObservatoryDir()
		var webCmd *exec.Cmd
		if _, err := os.Stat(filepath.Join(observatoryDir, "package.json")); err == nil {
			webCmd = exec.Command("npx", "next", "start", "-p", "3000", "-H", "0.0.0.0")
			webCmd.Dir = observatoryDir
			webCmd.Stdout = nil // suppress output
			webCmd.Stderr = nil
			if err := webCmd.Start(); err == nil {
				fmt.Fprintf(w, "  %s  %s\n", ui.Green("✓"), "Dashboard")
				fmt.Fprintf(w, "     %s\n", ui.Cyan("http://localhost:3000"))
				if lanIP := getLanIP(); lanIP != "" {
					fmt.Fprintf(w, "     %s\n", ui.Faint("http://"+lanIP+":3000"))
				}
			} else {
				fmt.Fprintf(w, "  %s  Dashboard  %s\n", ui.Yellow("⚠"), ui.Faint("not available (run: cd apps/dashboard && npm run build)"))
			}
		} else {
			// Try dev mode
			webCmd = exec.Command("npx", "next", "dev", "-p", "3000", "-H", "0.0.0.0")
			webCmd.Dir = observatoryDir
			webCmd.Stdout = nil
			webCmd.Stderr = nil
			if err := webCmd.Start(); err == nil {
				fmt.Fprintf(w, "  %s  Dashboard  %s\n", ui.Green("✓"), ui.Faint("(dev mode)"))
				fmt.Fprintf(w, "     %s\n", ui.Cyan("http://localhost:3000"))
			} else {
				fmt.Fprintf(w, "  %s  Dashboard  %s\n", ui.Yellow("⚠"), ui.Faint("not available"))
			}
		}

		fmt.Fprintln(w)

		// Start the Go API server
		store, err := universe.NewStore()
		if err != nil {
			return err
		}

		arch, archErr := universe.NewArchitectFromEnv()
		mode := ui.Green("full")
		if archErr != nil {
			mode = ui.Yellow("read-only") + ui.Faint(" (Docker not available)")
		}

		fmt.Fprintf(w, "  %s  API Server  %s\n", ui.Green("✓"), mode)
		fmt.Fprintf(w, "     %s\n", ui.Cyan("http://localhost:"+portFlag))
		fmt.Fprintln(w)

		// Footer
		fmt.Fprintf(w, "  %s\n", ui.Faint("Press Ctrl+C to stop"))
		fmt.Fprintln(w)

		// Handle graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Fprintf(w, "\n  %s\n\n", ui.Faint("Shutting down..."))
			if webCmd != nil && webCmd.Process != nil {
				_ = webCmd.Process.Kill()
			}
			os.Exit(0)
		}()

		srv := universe.NewObservatoryServer(store, arch, ":"+portFlag)
		return srv.Start()
	},
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open dashboard in browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		openCmd := exec.Command("open", "http://localhost:3000")
		return openCmd.Run()
	},
}

func init() {
	defaultDashHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(dashHelp)

	startCmd.Flags().StringVarP(&portFlag, "port", "p", "3001", "API server port")

	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(openCmd)
}

// findObservatoryDir locates the observatory Next.js app.
// Checks: relative to binary, relative to cwd, common dev paths.
func findObservatoryDir() string {
	candidates := []string{
		"apps/observatory",                                    // from repo root
		"../apps/observatory",                                 // from bin/
		filepath.Join(os.Getenv("HOME"), "Developer/spwn/apps/observatory"), // common dev path
	}
	// Also check relative to the binary location
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "apps", "observatory"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "package.json")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "apps/observatory" // fallback
}

// getLanIP returns the first non-loopback IPv4 address, or "" if none found.
func getLanIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}

func dashHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "dash" {
		if defaultDashHelp != nil {
			defaultDashHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ dash")+" "+ui.Faint("— visual dashboard"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "start", Desc: "Start the dashboard (API + web)"},
				{Name: "open", Desc: "Open dashboard in browser"},
			}},
		},
		"spwn dash [command]",
		"Use \"spwn dash <command> --help\" for more information.",
	)
}
