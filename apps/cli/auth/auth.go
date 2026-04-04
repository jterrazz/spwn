package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/foundation"
	"spwn.sh/core/foundation/auth"
	"github.com/spf13/cobra"
)

func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

// --- Parent command: spwn auth (shows status) ---

var defaultAuthHelp func(*cobra.Command, []string)

// Cmd is the auth command group.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials — login, logout, status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		jsonOut, _ := cmd.Flags().GetBool("json")

		// Resolve all providers via auth package
		creds := auth.ResolveAll()

		anthropic := creds[auth.ProviderAnthropic]
		openai := creds[auth.ProviderOpenAI]
		google := creds[auth.ProviderGoogle]

		if jsonOut {
			type providerJSON struct {
				Provider string `json:"provider"`
				OK       bool   `json:"ok"`
				Source   string `json:"source"`
				Type     string `json:"type"`
			}
			out := []providerJSON{
				{Provider: "anthropic", OK: anthropic.Type != auth.CredTypeNone, Source: anthropic.Source, Type: string(anthropic.Type)},
				{Provider: "openai", OK: openai.Type != auth.CredTypeNone, Source: openai.Source, Type: string(openai.Type)},
				{Provider: "google", OK: google.Type != auth.CredTypeNone, Source: google.Source, Type: string(google.Type)},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		s.Blank()

		// Table
		t := ui.NewTable(ui.ModeNormal, "PROVIDER", "STATUS", "SOURCE")
		t.AddRow("Anthropic", statusText(anthropic.Type != auth.CredTypeNone), anthropic.Source)
		t.AddRow("OpenAI", statusText(openai.Type != auth.CredTypeNone), openai.Source)
		t.AddRow("Google", statusText(google.Type != auth.CredTypeNone), google.Source)
		t.Render()

		// Token cache info
		s.Blank()
		cached := auth.ReadCachedToken()
		if cached != "" {
			tokenPath := foundation.BaseDir() + "/.auth-token"
			if info, err := os.Stat(tokenPath); err == nil {
				age := time.Since(info.ModTime())
				s.Info("Cached token:", fmt.Sprintf("%s (%s ago)", abbreviate(tokenPath), formatAge(age)))
			}
		} else {
			s.Info("Cached token:", ui.Faint("none"))
		}

		s.Blank()
		if anthropic.Type == auth.CredTypeNone {
			fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(`Run "spwn auth login" to set up credentials`))
			s.Blank()
		}

		return nil
	},
}

func init() {
	defaultAuthHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(authHelp)

	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(tokenCmd)
	Cmd.AddCommand(checkCmd)
}

func authHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "auth" {
		if defaultAuthHelp != nil {
			defaultAuthHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ auth")+" "+ui.Faint("— manage credentials"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "login", Desc: "Set up credentials (Keychain or manual)"},
				{Name: "logout", Desc: "Clear cached credentials"},
				{Name: "token <token>", Desc: "Set a token directly (CI / scripts)"},
				{Name: "check", Desc: "Validate credentials for all AI providers"},
			}},
			{Title: "Environment Variables", Commands: []ui.HelpEntry{
				{Name: "ANTHROPIC_API_KEY", Desc: "Anthropic Claude API key"},
				{Name: "OPENAI_API_KEY", Desc: "OpenAI API key"},
				{Name: "GOOGLE_API_KEY", Desc: "Google Gemini API key"},
			}},
		},
		"spwn auth              Show authentication status\n    spwn auth [command]",
		"",
	)
}

// --- spwn auth login ---

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Set up credentials — auto-detect from Keychain or paste manually",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		s.Blank()

		// 1. Try macOS Keychain via auth package
		s.Start("Checking Keychain...")
		cred := auth.Resolve(auth.ProviderAnthropic)
		if cred.Type == auth.CredTypeKeychain {
			// Cache it
			_ = auth.SaveToken(cred.Token)
			s.Done("Found subscription", "cached to "+abbreviate(foundation.BaseDir()+"/.auth-token"))
			s.Blank()
			s.Success("Authenticated.")
			s.Blank()
			return nil
		}
		s.Done("Keychain", ui.Faint("no Claude Code credentials found"))

		// 2. Check if already configured via env
		if cred.Type != auth.CredTypeNone {
			s.Done("Environment", cred.Source+" is set")
			s.Blank()
			s.Success("Authenticated via environment.")
			s.Blank()
			return nil
		}

		// 3. Manual input
		s.Blank()
		fmt.Fprint(os.Stderr, "  Paste your API key or OAuth token:\n  > ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return s.FailHint("Read failed", err, "Try setting ANTHROPIC_API_KEY instead")
		}
		input = strings.TrimSpace(input)
		if input == "" {
			return s.FailHint("Empty input", fmt.Errorf("no token provided"),
				"Paste an API key (sk-ant-...) or OAuth token")
		}

		// Detect type
		kind := "token"
		if strings.HasPrefix(input, "sk-ant-") {
			kind = "API key"
		}

		// Save
		if err := auth.SaveToken(input); err != nil {
			return s.FailHint("Save failed", err, "Check permissions on "+abbreviate(foundation.BaseDir()))
		}

		s.Blank()
		s.Done("Saved "+kind, abbreviate(foundation.BaseDir()+"/.auth-token"))
		s.Blank()
		s.Success("Authenticated.")
		s.Blank()
		return nil
	},
}

// --- spwn auth logout ---

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear cached credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		s.Blank()

		cached := auth.ReadCachedToken()
		if cached == "" {
			s.Info("No cached token", "nothing to clear")
			s.Blank()
			return nil
		}

		if err := auth.ClearToken(); err != nil {
			return s.FailHint("Clear failed", err, "Try removing manually: rm "+foundation.BaseDir()+"/.auth-token")
		}

		s.Done("Cleared cached token", abbreviate(foundation.BaseDir()+"/.auth-token"))

		// Warn about lingering env vars
		var active []string
		for _, key := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_AUTH_TOKEN"} {
			if os.Getenv(key) != "" {
				active = append(active, key)
			}
		}
		if len(active) > 0 {
			s.Blank()
			s.Warn("Environment", strings.Join(active, ", ")+" still set in your shell")
		}

		s.Blank()
		return nil
	},
}

// --- spwn auth token <token> ---

var tokenCmd = &cobra.Command{
	Use:   "token <token>",
	Short: "Set a token directly — for CI and scripts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		s.Blank()

		input := strings.TrimSpace(args[0])
		if input == "" {
			return s.FailHint("Empty token", fmt.Errorf("no token provided"), "Pass a non-empty token")
		}

		if err := auth.SaveToken(input); err != nil {
			return s.FailHint("Save failed", err, "Check permissions on "+abbreviate(foundation.BaseDir()))
		}

		kind := "token"
		if strings.HasPrefix(input, "sk-ant-") {
			kind = "API key"
		}

		s.Done("Saved "+kind, abbreviate(foundation.BaseDir()+"/.auth-token"))
		s.Blank()
		return nil
	},
}

// --- spwn auth check ---

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate credentials for all AI providers",
	RunE:  runCheck,
}

func runCheck(cmd *cobra.Command, _ []string) error {
	s := newStepper(cmd)
	jsonOut, _ := cmd.Flags().GetBool("json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.Blank()
	s.Start("Validating credentials...")

	results := auth.ValidateAll(ctx)

	if jsonOut {
		s.Done("Validation complete", "")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	s.Done("Validation complete", fmt.Sprintf("%d providers checked", len(results)))
	s.Blank()

	t := ui.NewTable(ui.ModeNormal, "PROVIDER", "STATUS", "TYPE", "SOURCE")
	for _, r := range results {
		status := ui.Green("✓") + " connected"
		if !r.Connected {
			if r.Error != "" {
				status = ui.Red("✗") + " " + r.Error
			} else {
				status = ui.Faint("○") + " not configured"
			}
		}
		t.AddRow(string(r.Provider), status, string(r.CredType), r.Source)
	}
	t.Render()

	// Show usage details for connected providers
	for _, r := range results {
		if r.Usage != nil && r.Connected {
			s.Blank()
			s.Info(string(r.Provider)+" usage:", "")
			if r.Usage.SessionPercent > 0 {
				s.Info("  Session:", fmt.Sprintf("%.1f%% used", r.Usage.SessionPercent))
			}
			if r.Usage.WeeklyPercent > 0 {
				s.Info("  Weekly:", fmt.Sprintf("%.1f%% used", r.Usage.WeeklyPercent))
			}
			if r.Usage.CreditsLimit > 0 {
				s.Info("  Credits:", fmt.Sprintf("%.2f / %.2f %s", r.Usage.CreditsUsed, r.Usage.CreditsLimit, r.Usage.Currency))
			}
		}
	}

	s.Blank()
	return nil
}

// --- Helpers ---

func statusText(ok bool) string {
	if ok {
		return ui.Green("\u2713") + " configured"
	}
	return ui.Faint("\u25CB") + " not set"
}

func abbreviate(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
