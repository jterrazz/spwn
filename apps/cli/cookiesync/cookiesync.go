// Package cookiesync hosts the `spwn cookie-sync ...` subcommands —
// the host-side counterpart to apps/spwn-cookie-sync, the browser
// extension that pushes session cookies to the gate.
//
// No pairing or secrets. The CLI just shows what the extension is
// doing (or would do once installed); persistence + per-element
// allowlists live entirely in the gate.
package cookiesync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

const gateURL = "http://127.0.0.1:9000"

// Cmd is the parent for `spwn cookie-sync …`.
var Cmd = &cobra.Command{
	Use:   "cookie-sync",
	Short: "Browser extension that auto-syncs session cookies to the gate (status + providers)",
	Long: `Browser extension that auto-syncs session cookies to the gate.

The extension watches your normal browser sessions on sites the gate
knows about (X today; LinkedIn etc. as elements are added) and pushes
the relevant session cookies to a locally-running spwn-gate. No
pairing, no secret — the gate listens on 127.0.0.1 only and accepts
just the cookie names each element declared, so other local processes
can't sneak unrelated cookies in.

Setup is two steps:

  1. spwn gate start                     # if not already running
  2. Open chrome://extensions/ → Developer mode → Load unpacked →
     select apps/spwn-cookie-sync/ in this repo

Then browse normally. The popup shows ● connected / ○ pending per
provider in real-time.`,
	RunE: runStatus,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show registered providers and per-provider last-sync timestamps",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List the providers the gate accepts cookie syncs for",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out)
		var providers []struct {
			Name    string   `json:"name"`
			Domains []string `json:"domains"`
			Cookies []string `json:"cookies"`
		}
		if err := getJSON("/sync/providers", &providers); err != nil {
			fmt.Fprintln(out, "  · gate not reachable on 127.0.0.1:9000")
			fmt.Fprintln(out, "    start it with: spwn gate start")
			fmt.Fprintln(out)
			return nil
		}
		fmt.Fprintf(out, "  %d provider(s) registered:\n", len(providers))
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

func runStatus(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)

	var body struct {
		Providers []struct {
			Name       string `json:"name"`
			LastSync   string `json:"last_sync"`
			HasCookies bool   `json:"has_cookies"`
		} `json:"providers"`
	}
	if err := getJSON("/sync/status", &body); err != nil {
		fmt.Fprintln(out, "  · gate not reachable on 127.0.0.1:9000")
		fmt.Fprintln(out, "    start it with: spwn gate start")
		fmt.Fprintln(out)
		return nil
	}

	if len(body.Providers) == 0 {
		fmt.Fprintln(out, "  · no providers registered (no gate elements use cookies yet)")
		fmt.Fprintln(out)
		return nil
	}

	fmt.Fprintln(out, "  ✓ gate reachable on 127.0.0.1:9000")
	fmt.Fprintln(out)
	for _, p := range body.Providers {
		state := "○ pending"
		detail := "no sync yet — visit the site once in your browser"
		if p.HasCookies || p.LastSync != "" {
			state = "● connected"
			if p.LastSync != "" {
				detail = "synced " + relTime(p.LastSync)
			} else {
				detail = "cookies on disk (gate restart since)"
			}
		}
		fmt.Fprintf(out, "    %-12s %s   %s\n", p.Name, state, detail)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  install the extension once: chrome://extensions/ → Developer mode → Load unpacked → apps/spwn-cookie-sync/")
	fmt.Fprintln(out)
	return nil
}

func relTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	d := time.Since(t)
	switch {
	case d < 5*time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func getJSON(path string, into any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", gateURL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(into)
}

func init() {
	Cmd.AddCommand(statusCmd, providersCmd)
}
