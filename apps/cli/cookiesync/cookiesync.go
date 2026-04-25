// Package cookiesync hosts the `spwn cookie-sync ...` subcommands —
// the host-side counterpart to apps/spwn-cookie-sync, the browser
// extension that pushes session cookies to the gate.
package cookiesync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/packages/gate"
)

// Cmd is the parent for `spwn cookie-sync …`. Wired into the root
// command in apps/cli/root.go.
var Cmd = &cobra.Command{
	Use:   "cookie-sync",
	Short: "Pair the browser extension that auto-syncs session cookies to the gate",
	Long: `Pair the spwn-cookie-sync browser extension.

The extension watches your normal browser sessions on allowlisted
sites (X, LinkedIn, …) and pushes the relevant cookies to the gate.
Spwn agents use those cookies to act as you on those sites — same
identity, same anti-detection profile, no OAuth needed.

Pairing is a one-time step:

  1. spwn cookie-sync register     — prints a secret SP-XXXX-XXXX-XXXX
  2. Install apps/spwn-cookie-sync/ in Chrome (Load unpacked)
  3. Click the extension icon → paste the secret → click Pair

After that, the extension and gate stay paired across reboots until
you run ` + "`spwn cookie-sync register --rotate`" + `.`,
}

var rotate bool

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Generate a new pairing secret for the browser extension",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if gate.HasSecret() && !rotate {
			fmt.Fprintln(cmd.OutOrStdout(), "  ✓ already paired (run with --rotate to invalidate the old secret)")
			fmt.Fprintln(cmd.OutOrStdout(), "    secret stored at:", gate.SecretPath())
			return nil
		}
		secret, err := gate.GenerateSecret()
		if err != nil {
			return fmt.Errorf("generate secret: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "  ✓ pairing secret generated")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "    "+secret)
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "  Install the extension (one time):")
		fmt.Fprintln(cmd.OutOrStdout(), "    1. Open chrome://extensions/ (or brave://extensions/, edge://extensions/)")
		fmt.Fprintln(cmd.OutOrStdout(), "    2. Toggle 'Developer mode' (top-right)")
		fmt.Fprintln(cmd.OutOrStdout(), "    3. Click 'Load unpacked' → select apps/spwn-cookie-sync/ in this repo")
		fmt.Fprintln(cmd.OutOrStdout(), "    4. Click the extension icon → paste the secret above → click Pair")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "  After pairing, browse normally — cookies sync invisibly to the gate.")
		fmt.Fprintln(cmd.OutOrStdout(), "  Run `spwn cookie-sync status` to confirm syncs are flowing.")
		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pairing state and per-provider last-sync timestamps",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out)
		if !gate.HasSecret() {
			fmt.Fprintln(out, "  · not paired   ·  spwn cookie-sync register")
			fmt.Fprintln(out)
			return nil
		}

		// Probe the gate via the same endpoint the extension uses, so
		// CLI status mirrors what the extension sees.
		secret, err := os.ReadFile(gate.SecretPath())
		if err != nil {
			return fmt.Errorf("read secret: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:9000/sync/status", nil)
		req.Header.Set("X-Spwn-Secret", string(secret))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(out, "  ✓ paired, but gate not reachable on 127.0.0.1:9000")
			fmt.Fprintln(out, "    start it with: spwn gate start")
			fmt.Fprintln(out)
			return nil
		}
		defer resp.Body.Close()
		var body struct {
			Providers []struct {
				Name     string `json:"name"`
				LastSync string `json:"last_sync"`
			} `json:"providers"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)

		fmt.Fprintln(out, "  ✓ paired with spwn-gate (127.0.0.1:9000)")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  providers:")
		for _, p := range body.Providers {
			when := p.LastSync
			if when == "" {
				when = "no sync yet — visit the site once in your browser"
			}
			fmt.Fprintf(out, "    %-12s %s\n", p.Name, when)
		}
		fmt.Fprintln(out)
		return nil
	},
}

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List the providers the gate accepts cookie syncs for",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:9000/sync/providers", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(out, "  · gate not reachable on 127.0.0.1:9000")
			fmt.Fprintln(out, "    start it with: spwn gate start")
			fmt.Fprintln(out)
			return nil
		}
		defer resp.Body.Close()
		var providers []gate.CookieProvider
		_ = json.NewDecoder(resp.Body).Decode(&providers)

		fmt.Fprintf(out, "  %d provider(s) configured:\n", len(providers))
		fmt.Fprintln(out)
		for _, p := range providers {
			fmt.Fprintf(out, "    %s\n", p.Name)
			fmt.Fprintf(out, "      domains:  %v\n", p.Domains)
			fmt.Fprintf(out, "      cookies:  %v\n", p.Cookies)
		}
		fmt.Fprintln(out)
		return nil
	},
}

var unpairCmd = &cobra.Command{
	Use:   "unpair",
	Short: "Delete the pairing secret (also removes per-provider cookies on disk)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		removed := false
		if err := os.Remove(gate.SecretPath()); err == nil {
			removed = true
		} else if !os.IsNotExist(err) {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintln(out)
		if removed {
			fmt.Fprintln(out, "  ✓ unpaired (extension can no longer push cookies)")
			fmt.Fprintln(out, "    note: cookies already on disk are kept; rm ~/.spwn/credentials/<provider>/cookies.json to clear them")
		} else {
			fmt.Fprintln(out, "  · already unpaired")
		}
		fmt.Fprintln(out)
		return nil
	},
}

func init() {
	registerCmd.Flags().BoolVar(&rotate, "rotate", false, "Generate a fresh secret even if one already exists; the extension will need re-pairing")
	Cmd.AddCommand(registerCmd, statusCmd, providersCmd, unpairCmd)
}
