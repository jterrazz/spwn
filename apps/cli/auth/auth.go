package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/auth"
	authgh "spwn.sh/packages/auth/gh"
	authgoogle "spwn.sh/packages/auth/google"
	authmcp "spwn.sh/packages/auth/mcp"
	"spwn.sh/packages/platform"
)

// Cmd is the auth command group. The bare invocation renders the
// Credential dashboard — auto-validates, shows every supported
// Method for every provider, surfaces the exact command to set or
// Refresh each one. Unknown positional args fail loudly so that
// Retired verbs (`status`, `check`, `token`) don't silently succeed
// With misleading output — users get an "unknown command" and see
// The real command list in help.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials — dashboard, login, use, logout, disable",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

var defaultAuthHelp func(*cobra.Command, []string)

func init() {
	defaultAuthHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(authHelp)

	Cmd.AddCommand(useCmd)
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(disableCmd)
	Cmd.AddCommand(enableCmd)
	Cmd.AddCommand(defaultCmd)

	// `spwn auth login <provider>` today only persists API keys —
	// OAuth is owned by the upstream tool (`claude login` /
	// `codex login`). The --oauth flag is retired: bare `spwn auth`
	// Already surfaces the "run claude login" hint when no OAuth
	// Credential is detected.
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "Save an API key for this provider")
	logoutCmd.Flags().StringVar(&logoutMethod, "method", "", "Scope logout to a single method (oauth | api_key)")
	defaultCmd.Flags().Bool("clear", false, "Remove the default preference (revert to auto-resolve)")
}

// providers is the set we render + accept as CLI args. Google is left
// off because it has no runtime wired in today; keeping it here would
// pollute the dashboard with a permanently-empty row.
var providers = []auth.Provider{auth.ProviderAnthropic, auth.ProviderOpenAI}

// methods is the fixed set of user-facing credential styles.
var methods = []auth.Method{auth.MethodOAuth, auth.MethodAPIKey}

// ── dashboard (bare `spwn auth`) ────────────────────────────────────

// runStatus renders the Extended-C credentials dashboard. For every
// Supported provider it emits one row per supported method (even if
// Unset) — each row reports what was detected, whether it validated
// Against the live API, and the next action: source+age for ✓ rows,
// Refresh hint for ✗ rows, setup command for · rows.
//
// Validations run in parallel with a 3-second budget. The 5-minute
// Positive-only cache (packages/auth/validate_cache.go) makes every
// Run after the first one instant.
//
// No file paths leak into user-visible text. Every "how to fix this"
// String comes from packages/auth/hints so the spawn pre-flight and
// This dashboard stay aligned.
func runStatus() error {
	w := os.Stderr
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Kick off every provider's methods in parallel. Each job returns
	// A single rendered row plus any follow-up action hint.
	type providerBlock struct {
		name    auth.Provider
		title   string
		rows    []dashboardRow
		active  string // human-readable active-method line when multi-valid
	}

	blocks := make([]providerBlock, 0, len(providers))
	for _, p := range providers {
		blocks = append(blocks, renderProviderBlock(ctx, p))
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s %s\n\n", ui.Cyan("⬡"), ui.Strong("Credentials"))
	for _, blk := range blocks {
		fmt.Fprintf(w, "  %s\n", ui.Strong(blk.title))
		for _, row := range blk.rows {
			renderDashboardRow(w, row)
		}
		if blk.active != "" {
			fmt.Fprintf(w, "      %s\n", ui.Faint(blk.active))
		}
		fmt.Fprintln(w)
	}

	// CLI tools (gh, …) — third bucket below "models" and "MCP".
	// CLI tools have their own auth shape (gh device-flow + a host
	// Hosts.yml) and a single row per tool covers it.
	renderCLISection(w)

	// MCP providers (Notion, …) — separate section because they're
	// A different kind of credential (OAuth token cached for an MCP
	// Server) than the AI-provider blocks above. We deliberately
	// Don't fold them in: users mentally separate "models" from
	// "Tools", and so should the dashboard.
	renderMCPSection(w)

	// Default provider footer. Faint label + bold value + faint
	// "change" hint, all on one line.
	def := auth.DefaultProvider()
	defText := "none set"
	if def != "" {
		defText = string(def)
	}
	fmt.Fprintf(w, "  %s %s  %s  %s\n\n",
		ui.Faint("Default:"),
		ui.Strong(defText),
		ui.Faint("·"),
		ui.Faint("spwn auth default <provider>"),
	)
	return nil
}

// renderCLISection prints the "Tools (CLI)" block — gh today, plus
// google (BYO-GCP, OAuth installed-app flow). Adding future CLIs is
// a one-liner once their auth package is wired.
func renderCLISection(w io.Writer) {
	fmt.Fprintf(w, "  %s\n", ui.Strong("Tools (CLI)"))

	gh := dashboardRow{method: "github"}
	if authgh.IsAuthenticated() {
		gh.glyph = ui.Green("✓")
		gh.detail = "file:" + abbreviate(authgh.CacheDir())
		gh.hintCommand = "spwn auth logout github"
	} else {
		gh.glyph = ui.Faint("·")
		gh.detail = "not set"
		gh.hintCommand = "spwn auth login github"
	}
	renderDashboardRow(w, gh)

	google := dashboardRow{method: "google"}
	if authgoogle.IsAuthenticated() {
		google.glyph = ui.Green("✓")
		google.detail = "file:" + abbreviate(authgoogle.CacheDir())
		google.hintCommand = "spwn auth logout google"
	} else {
		google.glyph = ui.Faint("·")
		google.detail = "not set"
		google.hintCommand = "spwn auth login google"
	}
	renderDashboardRow(w, google)

	fmt.Fprintln(w)
}

// renderMCPSection prints the "Tools (MCP)" block under the AI
// Provider blocks. One row per known MCP provider; the row says
// Whether tokens are cached and points at the right verb to set
// Or clear them.
func renderMCPSection(w io.Writer) {
	names := authmcp.Names()
	if len(names) == 0 {
		return
	}
	fmt.Fprintf(w, "  %s\n", ui.Strong("Tools (MCP)"))
	for _, name := range names {
		p, _ := authmcp.Lookup(name)
		row := dashboardRow{method: name}
		if authmcp.IsAuthenticated(p) {
			row.glyph = ui.Green("✓")
			row.detail = "file:" + abbreviate(authmcp.CacheDir())
			row.hintCommand = "spwn auth logout " + name
		} else {
			row.glyph = ui.Faint("·")
			row.detail = "not set"
			row.hintCommand = "spwn auth login " + name
		}
		renderDashboardRow(w, row)
	}
	fmt.Fprintln(w)
}

// dashboardRow is one rendered method line, fully resolved — the
// Renderer is pure formatting.
type dashboardRow struct {
	glyph       string // green ✓, red ✗, faint ·
	method      string // bold method name
	detail      string // right-hand "source · age" or "not set"
	hintCommand string // trailing action; rendered cyan (inline cmd)
}

// renderDashboardRow prints one method row at col 4.
func renderDashboardRow(w io.Writer, row dashboardRow) {
	// Method column is padded so the detail columns line up across
	// Rows. 14 chars is wide enough for "api_key"/"oauth" without
	// Hogging screen width.
	const methodCol = 14
	methodPadded := ui.PadVisible(ui.Strong(row.method), methodCol)
	line := fmt.Sprintf("    %s %s %s", row.glyph, methodPadded, ui.Faint(row.detail))
	if row.hintCommand != "" {
		line += "  " + ui.Faint("·") + "  " + ui.Cyan(row.hintCommand)
	}
	fmt.Fprintln(w, line)
}

// renderProviderBlock resolves + validates every supported method
// For one provider and returns the rendered rows plus any active-
// Method note. Parallelises the method probes so the whole block
// Finishes in max(one API call) rather than sum(all).
func renderProviderBlock(ctx context.Context, p auth.Provider) struct {
	name   auth.Provider
	title  string
	rows   []dashboardRow
	active string
} {
	result := struct {
		name   auth.Provider
		title  string
		rows   []dashboardRow
		active string
	}{name: p, title: providerTitle(p)}

	methods := auth.MethodCatalog(p)
	if len(methods) == 0 {
		return result
	}

	// Detect once so we see every source, not just the winner. The
	// Resolver's "pick one" logic is what the active: line names.
	detected := auth.DetectMethods(p)
	disabled := auth.IsProviderDisabled(p)
	byMethod := map[auth.Method]*auth.Credential{}
	for _, c := range detected {
		// First detection per method wins the row — mirrors the
		// Discovery order the resolver uses.
		if _, seen := byMethod[c.Method()]; !seen {
			byMethod[c.Method()] = c
		}
	}

	// Parallel validate. Cached positives come back instantly.
	type valResult struct {
		method auth.Method
		cred   *auth.Credential
		status *auth.ProviderStatus
	}
	results := make(chan valResult, len(methods))
	var wg sync.WaitGroup
	for _, m := range methods {
		cred := byMethod[auth.Method(m)]
		if cred == nil {
			// Nothing to validate — skip the goroutine.
			continue
		}
		wg.Add(1)
		go func(m auth.HintMethod, cred *auth.Credential) {
			defer wg.Done()
			results <- valResult{
				method: auth.Method(m),
				cred:   cred,
				status: auth.ValidateWithCache(ctx, cred, 5*time.Minute),
			}
		}(m, cred)
	}
	go func() { wg.Wait(); close(results) }()
	validations := map[auth.Method]valResult{}
	for r := range results {
		validations[r.method] = r
	}

	// Build one row per supported method.
	var validCreds []*auth.Credential
	for _, m := range methods {
		method := auth.Method(m)
		row := dashboardRow{method: string(method)}
		cred := byMethod[method]

		switch {
		case disabled:
			row.glyph = ui.Faint("·")
			row.detail = "disabled"
			row.hintCommand = "spwn auth enable " + string(p)
		case cred == nil:
			row.glyph = ui.Faint("·")
			row.detail = "not set"
			row.hintCommand = auth.NotSetHint(p, m)
		default:
			v, ok := validations[method]
			if ok && v.status != nil && v.status.Connected {
				row.glyph = ui.Green("✓")
				row.detail = credSourceDetail(cred)
				validCreds = append(validCreds, cred)
			} else {
				row.glyph = ui.Red("✗")
				reason := "rejected"
				if v.status != nil && v.status.Error != "" {
					reason = v.status.Error
				}
				row.detail = cred.Source + " · " + reason
				row.hintCommand = auth.RejectedHint(p, cred)
			}
		}
		result.rows = append(result.rows, row)
	}

	// Active note only when there's an ambiguity to resolve — 2+
	// Methods validated.
	if len(validCreds) >= 2 && !disabled {
		pref := auth.ActiveMethod(p)
		winner := validCreds[0]
		if pref != "" {
			for _, c := range validCreds {
				if c.Method() == pref {
					winner = c
					break
				}
			}
		}
		result.active = fmt.Sprintf("active: %s  ·  spwn auth use %s <method>", winner.Method(), p)
	}
	return result
}

// providerTitle returns the human-facing provider name ("Anthropic",
// "OpenAI") rather than the lowercase provider id. Small detail but
// The dashboard is the one place where branding is worth the cycles.
func providerTitle(p auth.Provider) string {
	switch p {
	case auth.ProviderAnthropic:
		return "Anthropic"
	case auth.ProviderOpenAI:
		return "OpenAI"
	}
	return string(p)
}

// credSourceDetail returns the right-hand detail for a ✓ row:
// "<source> · <age>" where source is the credential origin
// (keychain, env var name, …) and age is the cached validation age
// When available.
func credSourceDetail(cred *auth.Credential) string {
	if cred == nil {
		return ""
	}
	return cred.Source
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
  codex login    # then: spwn auth login openai

For hosted MCP providers (Notion, …), the command runs the OAuth dance
itself in a one-shot helper container — click the link it prints,
approve in your browser, the token persists for every world:

  spwn auth login notion`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// CLI-tool branch (gh) — host import or device-flow.
		if strings.EqualFold(args[0], "github") {
			return runGitHubLogin(cmd)
		}
		// Google branch — BYO-GCP wizard + installed-app PKCE flow.
		if strings.EqualFold(args[0], "google") {
			return runGoogleLogin(cmd)
		}
		// MCP providers branch — ref-by-name and own a different
		// storage path than AI providers.
		if mp, ok := authmcp.Lookup(args[0]); ok {
			return runMCPLogin(cmd, mp)
		}
		p, err := parseProvider(args[0])
		if err != nil {
			// If the user passed something that's neither an AI
			// provider, an MCP provider, nor a CLI-tool provider,
			// surface every list so the help is complete.
			return fmt.Errorf("%w (or one of the MCP providers: %s; or one of the CLI tools: github)", err, strings.Join(authmcp.Names(), ", "))
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
			s.Done(fmt.Sprintf("%s API key saved", p), "run `spwn auth` to confirm")
			s.Blank()
			return nil
		}

		// No flags: fall through to the same dashboard that bare
		// `spwn auth` renders. Users who reach for `spwn auth login
		// Anthropic` with no flag expecting a wizard get the
		// Self-explanatory method catalog instead, which surfaces
		// The exact commands for both OAuth (run claude login) and
		// API-key paths.
		_ = auth.SyncCredentials()
		return runStatus()
	},
}

// runGitHubLogin imports an existing host gh login (the happy path
// — most users already ran `gh auth login` once on their machine)
// or, in the future, drives a device-flow login when host gh isn't
// available. Either way the result is a plaintext hosts.yml under
// ~/.spwn/credentials/gh that bind-mounts cleanly into every world.
func runGitHubLogin(cmd *cobra.Command) error {
	s := ui.New()
	s.Blank()
	s.Info("Logging in to github", "importing host gh CLI credentials")
	s.Blank()
	err := authgh.Login(cmd.Context(), cmd.OutOrStdout())
	switch {
	case errors.Is(err, authgh.ErrGhNotInstalled):
		return s.FailHint("gh CLI not found on host", err,
			"Install gh first (e.g. `brew install gh`), run `gh auth login`, then re-run this command.")
	case errors.Is(err, authgh.ErrHostNotLoggedIn):
		return s.FailHint("host gh CLI is not logged in", err,
			"Run `gh auth login` on the host first (one-time, device-flow), then re-run this command.")
	case err != nil:
		return s.FailHint("github login failed", err,
			"Check ~/.config/gh permissions, then retry.")
	}
	s.Blank()
	s.Done("github tokens saved",
		fmt.Sprintf("ready — `gh ...` and `gh-mcp ...` inside any world will use these (cache: %s)", abbreviate(authgh.CacheDir())))
	s.Blank()
	return nil
}

// runMCPLogin drives `spwn auth login <mcp>`. The OAuth dance runs in
// a one-shot helper container; mcp2cli streams the "open this URL"
// hint to the user's terminal, waits for the redirect, exchanges the
// code, and persists tokens to ~/.spwn/credentials/mcp/.
func runMCPLogin(cmd *cobra.Command, p authmcp.Provider) error {
	s := ui.New()
	s.Blank()
	s.Info(fmt.Sprintf("Logging in to %s", p.Name), "a browser link will appear; click Allow, then this command exits")
	s.Blank()
	if err := authmcp.Login(cmd.Context(), p, cmd.OutOrStdout()); err != nil {
		return s.FailHint(fmt.Sprintf("%s login failed", p.Name), err,
			"Re-run `spwn auth login "+p.Name+"`. If the helper image build keeps failing, ensure Docker is running.")
	}
	s.Blank()
	s.Done(fmt.Sprintf("%s tokens saved", p.Name),
		fmt.Sprintf("ready — `notion-mcp ...` inside any world will use these tokens (cache: %s)", abbreviate(authmcp.CacheDir())))
	s.Blank()
	return nil
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

// runGoogleLogin walks the user through the one-time GCP setup the
// first time it's invoked, then runs Google's OAuth installed-app
// PKCE flow. Tokens land in ~/.spwn/credentials/google/, ready for
// the gate to consume via google.AccessToken on every gws call.
//
// Why an interactive wizard instead of `claude login`-style "go run
// this other tool first" — Google's restricted scopes (Gmail) make
// shipping a single OAuth client cost-prohibitive ($15k/yr CASA
// audit), so every spwn user runs their own OAuth client. The
// wizard keeps that ~10 min experience as low-friction as we can
// without paying for verification.
func runGoogleLogin(cmd *cobra.Command) error {
	s := ui.New()
	s.Blank()

	c, err := authgoogle.LoadClient()
	if err != nil {
		return s.FailHint("read client.json failed", err,
			"Check ~/.spwn/credentials/google/ permissions")
	}
	if c == nil {
		// First run: walk the user through GCP project setup, capture
		// their client_id/secret, persist before doing OAuth.
		c, err = authgoogle.PromptClient(cmd.InOrStdin(), cmd.OutOrStderr())
		if err != nil {
			return s.FailHint("OAuth client setup cancelled", err, "")
		}
		if err := authgoogle.SaveClient(c); err != nil {
			return s.FailHint("save client.json failed", err,
				"Check ~/.spwn/credentials/google/ is writable")
		}
	}

	s.Info("Logging in to google", "a browser tab will open; click Allow, then this command exits")
	s.Blank()

	if err := authgoogle.Login(cmd.Context(), c, cmd.OutOrStderr()); err != nil {
		return s.FailHint("google login failed", err,
			"Re-run `spwn auth login google`. If it loops, check that the OAuth consent screen has your email listed under Test users.")
	}
	s.Blank()
	s.Done("google tokens saved",
		fmt.Sprintf("ready — `gmail-mcp ...` and `gcal-mcp ...` inside any world will use these (cache: %s)", abbreviate(authgoogle.CacheDir())))
	s.Blank()
	return nil
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
		// CLI-tool branch (github).
		if strings.EqualFold(args[0], "github") {
			s := ui.New()
			s.Blank()
			if err := authgh.Logout(); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					s.Info("github", "already logged out")
					s.Blank()
					return nil
				}
				return s.FailHint("github logout failed", err,
					"Inspect "+abbreviate(authgh.CacheDir())+" by hand if this persists")
			}
			s.Done("Removed", "github tokens ("+abbreviate(authgh.HostsPath())+")")
			s.Blank()
			return nil
		}
		// MCP providers branch first, same shape as login.
		if mp, ok := authmcp.Lookup(args[0]); ok {
			s := ui.New()
			s.Blank()
			if err := authmcp.Logout(mp); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					s.Info(mp.Name, "already logged out")
					s.Blank()
					return nil
				}
				return s.FailHint(fmt.Sprintf("%s logout failed", mp.Name), err,
					"Inspect "+abbreviate(authmcp.CacheDir())+" by hand if this persists")
			}
			s.Done("Removed", mp.Name+" tokens ("+abbreviate(authmcp.ProviderTokenPath(mp))+")")
			s.Blank()
			return nil
		}
		p, err := parseProvider(args[0])
		if err != nil {
			return fmt.Errorf("%w (or one of the MCP providers: %s; or one of the CLI tools: github)", err, strings.Join(authmcp.Names(), ", "))
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
				{Name: "(bare)", Desc: "Live credential dashboard — auto-validates against each provider API"},
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
