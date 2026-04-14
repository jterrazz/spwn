// Package web implements `spwn web` - the local web UI.
//
// One command, no subcommands: it starts the Next.js frontend, the Go
// API server, and opens a browser tab. Ctrl+C tears everything down.
package web

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/world"
	"github.com/spf13/cobra"
)

var (
	portFlag string
	noOpen   bool
)

// Cmd is the `spwn web` command. It has no subcommands.
var Cmd = &cobra.Command{
	Use:   "web",
	Short: "Open the local web UI",
	Long: `Starts the Web UI (Next.js frontend + Go API server) and opens it
in your default browser. Blocks until Ctrl+C.`,
	RunE: runWeb,
}

func init() {
	Cmd.Flags().StringVarP(&portFlag, "port", "p", "3001", "API server port")
	Cmd.Flags().BoolVar(&noOpen, "no-open", false, "Don't open the browser")
}

func runWeb(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", ui.Strong("⬡ spwn web"))
	fmt.Fprintln(w)

	// Start the Next.js frontend in the background
	webDir := findWebDir()
	var webCmd *exec.Cmd
	webURL := "http://localhost:3000"

	if _, err := os.Stat(filepath.Join(webDir, "package.json")); err == nil {
		webCmd = exec.Command("npx", "next", "start", "-p", "3000", "-H", "0.0.0.0")
	} else {
		webCmd = exec.Command("npx", "next", "dev", "-p", "3000", "-H", "0.0.0.0")
	}
	webCmd.Dir = webDir
	webCmd.Stdout = nil
	webCmd.Stderr = nil

	if err := webCmd.Start(); err == nil {
		fmt.Fprintf(w, "  %s  %s\n", ui.Green("✓"), "Frontend")
		fmt.Fprintf(w, "     %s\n", ui.Cyan(webURL))
		if lanIP := getLanIP(); lanIP != "" {
			fmt.Fprintf(w, "     %s\n", ui.Faint("http://"+lanIP+":3000"))
		}
	} else {
		webCmd = nil
		fmt.Fprintf(w, "  %s  Frontend  %s\n", ui.Yellow("⚠"), ui.Faint("not available (run: cd apps/web && npm install)"))
	}

	fmt.Fprintln(w)

	// Start the Go API server
	store, err := world.NewStore()
	if err != nil {
		return err
	}

	arch, archErr := world.NewArchitectFromEnv()
	mode := ui.Green("full")
	if archErr != nil {
		mode = ui.Yellow("read-only") + ui.Faint(" (Docker not available)")
	}

	fmt.Fprintf(w, "  %s  API Server  %s\n", ui.Green("✓"), mode)
	fmt.Fprintf(w, "     %s\n", ui.Cyan("http://localhost:"+portFlag))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %s\n", ui.Faint("Press Ctrl+C to stop"))
	fmt.Fprintln(w)

	// Graceful shutdown: kill the frontend when we exit
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

	// Open browser after a small delay so the frontend has a moment to bind.
	if !noOpen && webCmd != nil {
		go func() {
			time.Sleep(750 * time.Millisecond)
			_ = openBrowser(webURL)
		}()
	}

	srv := world.NewAPIServer(store, arch, ":"+portFlag)
	return srv.Start()
}

// openBrowser opens url in the default browser on macOS, Linux, or Windows.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// findWebDir locates the web UI Next.js app.
func findWebDir() string {
	candidates := []string{
		"apps/web",
		"../apps/web",
		filepath.Join(os.Getenv("HOME"), "Developer/spwn/apps/web"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "apps", "web"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "package.json")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "apps/web"
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
