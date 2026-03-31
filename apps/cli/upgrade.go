package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade spwn to the latest version",
	Long: `Downloads and installs the latest spwn release from GitHub.

Detects your OS and architecture, downloads the matching binary,
and replaces the current installation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New(quiet, verbose, jsonOutput)
		s.Blank()

		// Detect current version
		s.Info("Current version:", Version)

		// Check latest version from GitHub API
		s.Start("Checking for updates...")
		latest, err := getLatestVersion()
		if err != nil {
			s.Fail("Check failed", err)
			return err
		}

		if latest == Version || latest == "v"+Version {
			s.Done("Already up to date", Version)
			s.Blank()
			return nil
		}

		s.Done("New version available", latest)

		// Detect OS/arch
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		filename := fmt.Sprintf("spwn_%s_%s.tar.gz", goos, goarch)
		url := fmt.Sprintf("https://github.com/jterrazz/spwn/releases/download/%s/%s", latest, filename)

		// Download
		s.Start("Downloading " + latest + "...")
		tmpDir, err := os.MkdirTemp("", "spwn-upgrade-")
		if err != nil {
			return fmt.Errorf("error: cannot create temp dir.\n%w", err)
		}
		defer os.RemoveAll(tmpDir)

		tarPath := tmpDir + "/" + filename
		dlCmd := exec.Command("curl", "-fsSL", url, "-o", tarPath)
		if output, err := dlCmd.CombinedOutput(); err != nil {
			s.Fail("Download failed", fmt.Errorf("%s", string(output)))
			return fmt.Errorf("error: download failed.\n%w", err)
		}
		s.Done("Downloaded", filename)

		// Extract
		s.Start("Extracting...")
		extractCmd := exec.Command("tar", "-xzf", tarPath, "-C", tmpDir)
		if output, err := extractCmd.CombinedOutput(); err != nil {
			s.Fail("Extract failed", fmt.Errorf("%s", string(output)))
			return fmt.Errorf("error: extract failed.\n%w", err)
		}

		// Find current binary path
		currentBin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error: cannot find current binary.\n%w", err)
		}

		// Replace
		s.Start("Installing...")
		newBin := tmpDir + "/spwn"
		if _, err := os.Stat(newBin); os.IsNotExist(err) {
			return fmt.Errorf("error: extracted binary not found at %s", newBin)
		}

		// Copy new binary over current (using cp to preserve permissions)
		cpCmd := exec.Command("cp", newBin, currentBin)
		if output, err := cpCmd.CombinedOutput(); err != nil {
			// Try with sudo
			sudoCmd := exec.Command("sudo", "cp", newBin, currentBin)
			sudoCmd.Stdin = os.Stdin
			if sudoOutput, sudoErr := sudoCmd.CombinedOutput(); sudoErr != nil {
				s.Fail("Install failed", fmt.Errorf("%s\n%s", string(output), string(sudoOutput)))
				return fmt.Errorf("error: cannot replace binary.\n%w", sudoErr)
			}
		}

		s.Done("Upgraded", fmt.Sprintf("%s → %s", Version, latest))
		s.Blank()
		return nil
	},
}

// getLatestVersion fetches the latest release tag from GitHub.
func getLatestVersion() (string, error) {
	cmd := exec.Command("curl", "-fsSL", "https://api.github.com/repos/jterrazz/spwn/releases/latest")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cannot reach GitHub API: %w", err)
	}

	// Simple JSON parsing without importing encoding/json
	str := string(output)
	idx := strings.Index(str, `"tag_name"`)
	if idx == -1 {
		return "", fmt.Errorf("no tag_name in response")
	}

	// Find the value after "tag_name": "
	rest := str[idx+len(`"tag_name"`):]
	start := strings.Index(rest, `"`) + 1
	end := strings.Index(rest[start:], `"`)
	if start < 1 || end < 0 {
		return "", fmt.Errorf("cannot parse tag_name")
	}

	return rest[start : start+end], nil
}
