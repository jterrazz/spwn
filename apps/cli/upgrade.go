package cli

import (
	"context"
	"fmt"
	"os"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/base/update"
	"spwn.sh/packages/world"

	"github.com/spf13/cobra"
)

var (
	upgradeChannel string
	upgradeCheck   bool
	upgradeForce   bool
)

func init() {
	upgradeCmd.Flags().StringVar(&upgradeChannel, "channel", "stable", "Release channel: stable or beta")
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "Check for updates but do not install")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "Install even if already up to date")
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade spwn to the latest version",
	Long: `Downloads and installs the latest spwn release from GitHub.

Fetches the release binary for your OS/architecture, verifies its SHA256
against the checksums published with the release, and atomically replaces
the current binary. Running worlds are stopped gracefully before the swap.`,
	Example: `  spwn upgrade              # install the latest stable release
  spwn upgrade --check      # just check, don't install
  spwn upgrade --channel beta
  spwn upgrade --force      # reinstall current version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New()
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		s.Blank()
		s.Info("Current version:", Version)

		client := &update.GitHubClient{Owner: "jterrazz", Repo: "spwn"}

		s.Start("Checking for updates...")
		plan, err := update.CheckForUpdate(ctx, client, Version, update.CheckOpts{
			Channel: update.Channel(upgradeChannel),
		})
		if err != nil {
			return s.FailHint("Check failed", err, "Verify your internet connection and try again")
		}

		if !plan.UpdateAvail && !upgradeForce {
			s.Done("Already up to date", Version)
			s.Blank()
			return nil
		}

		s.Done("Latest version", plan.Latest.String())

		if upgradeCheck {
			if plan.UpdateAvail {
				s.Info("Update available:", fmt.Sprintf("%s → %s", Version, plan.Latest.String()))
				s.Info("Release notes:", plan.Release.HTMLURL)
				s.Blank()
				s.Info("Run to install:", "spwn upgrade")
				s.Blank()
			}
			return nil
		}

		// Pre-flight: find the platform asset before stopping anything.
		if plan.Asset == nil {
			return fmt.Errorf("no release asset found for %s in %s", plan.Platform, plan.Latest.String())
		}

		// Stop running worlds + architect daemon before swapping the binary.
		if err := stopSpwnWorkloads(ctx, s); err != nil {
			s.Warn("Warning", fmt.Sprintf("proceeding despite: %v", err))
		}

		// Figure out where our own binary lives.
		targetPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate current binary: %w", err)
		}

		err = update.Apply(ctx, plan, update.ApplyOpts{
			BinaryName: "spwn",
			TargetPath: targetPath,
			Progress: func(msg string) {
				s.Start(msg + "...")
			},
		})
		if err != nil {
			return s.FailHint("Install failed", err, "Check permissions on "+targetPath)
		}
		s.Done("Upgraded", fmt.Sprintf("%s → %s", Version, plan.Latest.String()))
		s.Info("Release notes:", plan.Release.HTMLURL)
		s.Blank()
		return nil
	},
}

// stopSpwnWorkloads gracefully stops any running worlds and the architect
// daemon before we swap the binary. Errors are logged but not fatal - the
// upgrade should proceed even if Docker is offline.
func stopSpwnWorkloads(ctx context.Context, s *ui.Stepper) error {
	arc, err := world.NewArchitectFromEnv()
	if err != nil {
		return nil // no Docker, nothing to stop
	}
	worlds, err := arc.List(ctx)
	if err != nil {
		return err
	}
	for _, w := range worlds {
		s.Start("Stopping " + w.ID + "...")
		if _, err := arc.Destroy(ctx, w.ID); err != nil {
			s.Warn("Warning", fmt.Sprintf("failed to stop %s: %v", w.ID, err))
			continue
		}
		label := w.ID
		if w.Agent != "" {
			label += " (" + w.Agent + ")"
		}
		s.Done("Stopped", label)
	}
	if info, statusErr := world.GetArchitectDaemonStatus(ctx); statusErr == nil && info.Running {
		s.Start("Stopping architect...")
		if stopErr := world.StopArchitectDaemon(ctx); stopErr != nil {
			s.Warn("Warning", fmt.Sprintf("failed to stop architect: %v", stopErr))
		} else {
			s.Done("Architect stopped", "")
		}
	}
	return nil
}
