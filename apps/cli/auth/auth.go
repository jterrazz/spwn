package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/auth"
	"spwn.sh/packages/platform"
)

// Cmd is the auth command group. The bare invocation renders the
// multi-method dashboard so users see every credential spwn detected,
// which one is active, and every disable/use/logout action available.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials — status, login, use, logout, disable",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

var defaultAuthHelp func(*cobra.Command, []string)

func init() {
	defaultAuthHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(authHelp)

	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(useCmd)
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(disableCmd)
	Cmd.AddCommand(enableCmd)
	Cmd.AddCommand(defaultCmd)
	Cmd.AddCommand(checkCmd)
	Cmd.AddCommand(tokenCmd) // deprecated alias for login anthropic --api-key

	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "Save an API key for this provider")
	loginCmd.Flags().BoolVar(&loginOAuth, "oauth", false, "Print OAuth login instructions for this provider")
	logoutCmd.Flags().StringVar(&logoutMethod, "method", "", "Scope logout to a single method (oauth | api_key)")
	defaultCmd.Flags().Bool("clear", false, "Remove the default preference (revert to auto-resolve)")
}

// providers is the set we render + accept as CLI args. Google is left
// off because it has no runtime wired in today; keeping it here would
// pollute the dashboard with a permanently-empty row.
var providers = []auth.Provider{auth.ProviderAnthropic, auth.ProviderOpenAI}

// methods is the fixed set of user-facing credential styles.
var methods = []auth.Method{auth.MethodOAuth, auth.MethodAPIKey}

// ── status / dashboard ──────────────────────────────────────────────

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show every detected credential and which one is active",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

// runStatus renders a row per detected (provider, method) pair so
// users can see what spwn found, what won, and why. Providers with
// no creds still get a single "missing" row — the dashboard must
// always show every provider, not just the configured ones.
func runStatus() error {
	s := ui.New()
	s.Blank()

	t := ui.NewTable("PROVIDER", "METHOD", "STATE", "SOURCE")
	for _, p := range providers {
		disabled := auth.IsProviderDisabled(p)
		activeMethod := auth.ActiveMethod(p)
		detected := auth.DetectMethods(p)

		if len(detected) == 0 {
			state := "missing"
			if disabled {
				state = "disabled"
			}
			t.AddRow(string(p), "—", state, "—")
			continue
		}

		// Figure out which single detection is "active" — the one
		// Resolve would return right now. pickByPref logic is
		// reproduced here so the dashboard doesn't disagree with
		// the runtime.
		winner := pickActive(p, detected, activeMethod, disabled)
		for _, cred := range detected {
			state := "known"
			switch {
			case disabled:
				state = "disabled"
			case cred == winner:
				state = "active"
			}
			t.AddRow(string(p), string(cred.Method()), state, cred.Source)
		}
	}
	t.Render()

	// Surface the default provider right under the table so users can
	// See it at a glance before reaching for the command list.
	if def := auth.DefaultProvider(); def != "" {
		s.Blank()
		s.Info("default provider:", string(def))
	}

	// Hints — keep them short and action-oriented. The dashboard
	// Should teach by example, not blog.
	s.Blank()
	s.Info("Pick a method:", "spwn auth use <provider> <oauth|api_key>")
	s.Info("Pick a default:", "spwn auth default <provider>")
	s.Info("Log out cleanly:", "spwn auth logout <provider>")
	s.Info("Opt out entirely:", "spwn auth disable <provider>")
	s.Blank()
	return nil
}

// pickActive mirrors auth.pickByPref's selection logic for the
// dashboard. Returns nil when no credential would be active (disabled
// provider, empty detection list).
func pickActive(p auth.Provider, detected []*auth.Credential, pref auth.Method, disabled bool) *auth.Credential {
	if disabled || len(detected) == 0 {
		return nil
	}
	if pref != "" {
		for _, c := range detected {
			if c.Method() == pref {
				return c
			}
		}
	}
	return detected[0]
}

// ── use ─────────────────────────────────────────────────────────────

var useCmd = &cobra.Command{
	Use:   "use <provider> <method>",
	Short: "Pick which credential method spwn should prefer",
	Long: `Flip the active method for a provider. Run without a method
to clear the preference (back to auto-select).

Example:
  spwn auth use anthropic oauth
  spwn auth use openai api_key`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		var m auth.Method
		if len(args) == 2 {
			m, err = parseMethod(args[1])
			if err != nil {
				return err
			}
		}
		if err := auth.SetActiveMethod(p, m); err != nil {
			return err
		}
		s := ui.New()
		s.Blank()
		if m == "" {
			s.Success(fmt.Sprintf("%s: auto-select restored", p))
		} else {
			s.Success(fmt.Sprintf("%s: active method set to %s", p, m))
		}
		s.Blank()
		return runStatus()
	},
}

// ── login ───────────────────────────────────────────────────────────

var (
	loginAPIKey string
	loginOAuth  bool
)

var loginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Set up credentials for a provider",
	Long: `Register credentials for a provider. The simplest path is an
API key:

  spwn auth login anthropic --api-key sk-ant-...

For OAuth-backed subscription access (Claude.ai / ChatGPT Plus via codex),
run the upstream CLI login first, then re-run this command — spwn will
detect the new credential and record it:

  claude login   # then: spwn auth login anthropic
  codex login    # then: spwn auth login openai`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		s := ui.New()
		s.Blank()

		// Explicit API-key path: persist into the provider's cache.
		if loginAPIKey != "" {
			if err := saveAPIKey(p, strings.TrimSpace(loginAPIKey)); err != nil {
				return s.FailHint("Save failed", err,
					"Check permissions on "+abbreviate(platform.BaseDir()))
			}
			_ = auth.SyncCredentials()
			s.Done(fmt.Sprintf("%s API key saved", p), "spwn auth status to confirm")
			s.Blank()
			return nil
		}

		// OAuth: we don't own the flow (yet). Point at the upstream CLI
		// and then re-resolve so the user sees the result in-line.
		if loginOAuth {
			s.Info("OAuth login", "run the runtime CLI's login flow:")
			switch p {
			case auth.ProviderAnthropic:
				fmt.Fprintln(os.Stderr, "    claude login")
			case auth.ProviderOpenAI:
				fmt.Fprintln(os.Stderr, "    codex login")
			}
			s.Blank()
			s.Info("Then verify:", "spwn auth status")
			s.Blank()
			return nil
		}

		// No flags: detect and report what spwn can see. Keeps the
		// old verb's discovery behaviour reachable.
		detected := auth.DetectMethods(p)
		if len(detected) == 0 {
			return s.FailHint(fmt.Sprintf("%s not configured", p),
				errors.New("no credentials detected"),
				fmt.Sprintf("Run `spwn auth login %s --api-key <key>` or `--oauth` for instructions", p))
		}
		for _, cred := range detected {
			s.Done(fmt.Sprintf("%s: %s", p, cred.Method()), cred.Source)
		}
		_ = auth.SyncCredentials()
		s.Blank()
		return nil
	},
}

// saveAPIKey persists a user-entered API key into the right per-provider
// slot. Anthropic reuses the legacy `.auth-token` cache so downstream
// tools that read it directly (Claude CLI, VS Code extension) keep
// working. OpenAI needs a net-new cache since `~/.codex/auth.json` is
// OAuth-shaped and owned by codex; we drop ours under CredentialsDir.
func saveAPIKey(p auth.Provider, key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	switch p {
	case auth.ProviderAnthropic:
		return auth.SaveToken(key)
	case auth.ProviderOpenAI:
		return errors.New("OpenAI API key persistence not yet supported; export OPENAI_API_KEY in your shell for now")
	}
	return fmt.Errorf("unsupported provider %q", p)
}

// ── logout ──────────────────────────────────────────────────────────

var logoutMethod string

var logoutCmd = &cobra.Command{
	Use:   "logout <provider>",
	Short: "Clear stored credentials for a provider",
	Long: `Remove every stored credential for a provider — cache file,
macOS keychain entry, runtime-CLI auth files. Does NOT unset env vars
(the shell owns those); any active env vars are surfaced so you know
to unset them manually.

  spwn auth logout anthropic
  spwn auth logout openai
  spwn auth logout anthropic --method api_key   # spare keychain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		opts := auth.LogoutOpts{}
		if logoutMethod != "" {
			opts.Method, err = parseMethod(logoutMethod)
			if err != nil {
				return err
			}
		}
		s := ui.New()
		s.Blank()

		res := auth.LogoutProvider(p, opts)

		if len(res.Cleared) == 0 && len(res.Remaining) == 0 && !res.HasErrors() {
			s.Info(string(p), "already logged out")
			s.Blank()
			return nil
		}
		for _, item := range res.Cleared {
			s.Done("Removed", item)
		}
		for _, item := range res.Remaining {
			s.Warn("Still set (shell env)", item+"  — run `unset "+strings.TrimPrefix(item, "env:")+"` to clear")
		}
		for _, err := range res.Errors {
			s.Warn("Error", err.Error())
		}
		s.Blank()
		if res.HasErrors() {
			return errors.New("logout completed with errors")
		}
		return nil
	},
}

// ── disable / enable ────────────────────────────────────────────────

var disableCmd = &cobra.Command{
	Use:   "disable <provider>",
	Short: "Tell spwn not to use this provider, even if creds exist",
	Long: `Opt a provider out without touching credentials. Useful when
you want spwn to ignore (say) codex OAuth on your machine but leave
the codex CLI working.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		if err := auth.DisableProvider(p); err != nil {
			return err
		}
		s := ui.New()
		s.Blank()
		s.Success(fmt.Sprintf("%s disabled; spwn will behave as though it has no credentials", p))
		s.Info("Re-enable with:", fmt.Sprintf("spwn auth enable %s", p))
		s.Blank()
		return nil
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable <provider>",
	Short: "Reverse a previous `disable`",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		if err := auth.EnableProvider(p); err != nil {
			return err
		}
		s := ui.New()
		s.Blank()
		s.Success(fmt.Sprintf("%s enabled", p))
		s.Blank()
		return runStatus()
	},
}

// ── default ─────────────────────────────────────────────────────────

var defaultCmd = &cobra.Command{
	Use:   "default [provider]",
	Short: "Pick which provider spwn prefers when multiple are authenticated",
	Long: `Set a soft preference for which provider's runtime spwn picks
when you're logged into more than one and no runtime is pinned at the
project or agent level.

This is the durable answer to the "multiple providers authenticated
and no runtime pinned" error — set it once and spwn will quietly
resolve ambiguity in that provider's favour. Does NOT disable the
other provider or override agent.yaml / spwn.yaml pins.

Example:
  spwn auth default anthropic        # prefer claude-code
  spwn auth default --clear          # revert to auto-resolve`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clear, _ := cmd.Flags().GetBool("clear")
		s := ui.New()

		if clear {
			if err := auth.SetDefaultProvider(""); err != nil {
				return err
			}
			s.Blank()
			s.Success("default provider cleared")
			s.Blank()
			return nil
		}

		if len(args) == 0 {
			current := auth.DefaultProvider()
			s.Blank()
			if current == "" {
				s.Info("default provider:", "not set (auto-resolve)")
				s.Info("Set one with:", "spwn auth default <provider>")
			} else {
				s.Info("default provider:", string(current))
				s.Info("Clear with:", "spwn auth default --clear")
			}
			s.Blank()
			return nil
		}

		p, err := parseProvider(args[0])
		if err != nil {
			return err
		}
		if auth.IsProviderDisabled(p) {
			return s.FailHint("Default refused",
				fmt.Errorf("%s is currently disabled", p),
				fmt.Sprintf("Run `spwn auth enable %s` first, or pick a different provider", p))
		}
		if err := auth.SetDefaultProvider(p); err != nil {
			return err
		}
		s.Blank()
		s.Success(fmt.Sprintf("default provider set to %s", p))
		s.Blank()
		return nil
	},
}

// ── check ───────────────────────────────────────────────────────────

var checkCmd = &cobra.Command{
	Use:   "check [provider]",
	Short: "Validate active credentials against each provider's API",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.Blank()
		s.Start("Validating credentials...")

		var results []auth.ProviderStatus
		if len(args) == 1 {
			p, err := parseProvider(args[0])
			if err != nil {
				return err
			}
			cred := auth.Resolve(p)
			results = append(results, *auth.Validate(ctx, cred))
		} else {
			results = auth.ValidateAll(ctx)
		}

		s.Done("Validation complete", fmt.Sprintf("%d provider(s)", len(results)))
		s.Blank()

		t := ui.NewTable("PROVIDER", "STATUS", "METHOD", "SOURCE")
		for _, r := range results {
			var status string
			switch {
			case r.Connected:
				status = "connected"
			case r.Error != "":
				status = ui.Red("✗") + " " + r.Error
			default:
				status = "not configured"
			}
			method := string((&auth.Credential{Type: r.CredType}).Method())
			if method == "" {
				method = "—"
			}
			t.AddRow(string(r.Provider), status, method, r.Source)
		}
		t.Render()

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
	},
}

// ── token (deprecated) ──────────────────────────────────────────────

var tokenCmd = &cobra.Command{
	Use:    "token <token>",
	Short:  "Deprecated — use `login anthropic --api-key <token>` instead",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		loginAPIKey = args[0]
		return loginCmd.RunE(cmd, []string{"anthropic"})
	},
}

// ── help ────────────────────────────────────────────────────────────

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
			{Title: "Inspect", Commands: []ui.HelpEntry{
				{Name: "status", Desc: "Show every detected credential (default when no subcommand)"},
				{Name: "check [provider]", Desc: "Validate active credentials against each API"},
			}},
			{Title: "Configure", Commands: []ui.HelpEntry{
				{Name: "use <provider> <method>", Desc: "Pick oauth or api_key for a provider"},
				{Name: "login <provider>", Desc: "Register credentials (--api-key <val> or --oauth)"},
				{Name: "default <provider>", Desc: "Prefer this provider when multiple are authenticated"},
				{Name: "disable <provider>", Desc: "Ignore a provider without deleting creds"},
				{Name: "enable <provider>", Desc: "Reverse a previous disable"},
			}},
			{Title: "Remove", Commands: []ui.HelpEntry{
				{Name: "logout <provider>", Desc: "Clear all credentials for a provider (keychain, files)"},
			}},
			{Title: "Supported providers", Commands: []ui.HelpEntry{
				{Name: "anthropic", Desc: "Claude — OAuth (subscription) or API key"},
				{Name: "openai", Desc: "Codex — OAuth (ChatGPT Plus) or API key (env-only today)"},
			}},
		},
		"spwn auth [command]",
		"",
	)
}

// ── shared helpers ──────────────────────────────────────────────────

func parseProvider(raw string) (auth.Provider, error) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	for _, p := range providers {
		if string(p) == lower {
			return p, nil
		}
	}
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = string(p)
	}
	sort.Strings(names)
	return "", fmt.Errorf("unknown provider %q; try one of: %s", raw, strings.Join(names, ", "))
}

func parseMethod(raw string) (auth.Method, error) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	for _, m := range methods {
		if string(m) == lower {
			return m, nil
		}
	}
	names := make([]string, len(methods))
	for i, m := range methods {
		names[i] = string(m)
	}
	sort.Strings(names)
	return "", fmt.Errorf("unknown method %q; try one of: %s", raw, strings.Join(names, ", "))
}

func abbreviate(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
