package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/base"
	"spwn.sh/packages/base/auth"
	"github.com/spf13/cobra"
)

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
}

// --- Parent command: spwn auth (shows status) ---

var defaultAuthHelp func(*cobra.Command, []string)

// Cmd is the auth command group.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials - login, logout, status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)

		// Resolve all providers via auth package
		creds := auth.ResolveAll()

		s.Blank()

		// Table
		t := ui.NewTable("PROVIDER", "STATUS", "SOURCE")
		for _, p := range []auth.Provider{auth.ProviderAnthropic, auth.ProviderOpenAI} {
			cred := creds[p]
			t.AddRow(string(p), statusText(cred.Type != auth.CredTypeNone), cred.Source)
		}
		t.Render()

		// Token cache info
		s.Blank()
		cached := auth.ReadCachedToken()
		if cached != "" {
			tokenPath := base.BaseDir() + "/.auth-token"
			if info, err := os.Stat(tokenPath); err == nil {
				age := time.Since(info.ModTime())
				s.Info("Cached token:", fmt.Sprintf("%s (%s ago)", abbreviate(tokenPath), formatAge(age)))
			}
		} else {
			s.Info("Cached token:", ui.Faint("none"))
		}

		s.Blank()
		anyConfigured := false
		for _, cred := range creds {
			if cred.Type != auth.CredTypeNone {
				anyConfigured = true
				break
			}
		}
		if !anyConfigured {
			fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint(`Sign in: "claude login" (Anthropic) or "codex login" (OpenAI)`))
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
		ui.Strong("⬡ auth")+" "+ui.Faint("- manage credentials"),
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
			}},
		},
		"spwn auth              Show authentication status\n    spwn auth [command]",
		"",
	)
}

// --- spwn auth login ---

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Detect credentials from CLI logins and system keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		s.Blank()

		found := false

		// Anthropic: check keychain (Claude Code subscription)
		s.Start("Checking Anthropic...")
		cred := auth.Resolve(auth.ProviderAnthropic)
		if cred.Type != auth.CredTypeNone {
			if cred.Type == auth.CredTypeKeychain {
				_ = auth.SaveToken(cred.Token)
			}
			_ = auth.EnableProvider(auth.ProviderAnthropic)
			s.Done("Anthropic", cred.Source)
			found = true
		} else {
			s.Done("Anthropic", ui.Faint("not found - run: claude login"))
		}

		// OpenAI: check codex auth.json
		s.Start("Checking OpenAI...")
		openai := auth.Resolve(auth.ProviderOpenAI)
		if openai.Type != auth.CredTypeNone {
			_ = auth.EnableProvider(auth.ProviderOpenAI)
			s.Done("OpenAI", openai.Source)
			found = true
		} else {
			s.Done("OpenAI", ui.Faint("not found - run: codex login"))
		}

		// Sync credentials
		_ = auth.SyncCredentials()

		s.Blank()
		if found {
			s.Success("Credentials synced.")
		} else {
			fmt.Fprintf(os.Stderr, "  %s\n", ui.Faint("Sign in with your runtime CLI first:"))
			fmt.Fprintf(os.Stderr, "    %s\n", "claude login    (Anthropic subscription)")
			fmt.Fprintf(os.Stderr, "    %s\n", "codex login     (OpenAI subscription)")
		}
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
			return s.FailHint("Clear failed", err, "Try removing manually: rm "+base.BaseDir()+"/.auth-token")
		}

		s.Done("Cleared cached token", abbreviate(base.BaseDir()+"/.auth-token"))

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
	Short: "Set a token directly - for CI and scripts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)
		s.Blank()

		input := strings.TrimSpace(args[0])
		if input == "" {
			return s.FailHint("Empty token", fmt.Errorf("no token provided"), "Pass a non-empty token")
		}

		if err := auth.SaveToken(input); err != nil {
			return s.FailHint("Save failed", err, "Check permissions on "+abbreviate(base.BaseDir()))
		}

		kind := "token"
		if strings.HasPrefix(input, "sk-ant-") {
			kind = "API key"
		}

		s.Done("Saved "+kind, abbreviate(base.BaseDir()+"/.auth-token"))
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.Blank()
	s.Start("Validating credentials...")

	results := auth.ValidateAll(ctx)

	s.Done("Validation complete", fmt.Sprintf("%d providers checked", len(results)))
	s.Blank()

	t := ui.NewTable("PROVIDER", "STATUS", "TYPE", "SOURCE")
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
