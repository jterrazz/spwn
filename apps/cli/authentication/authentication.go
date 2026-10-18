package authentication

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
	"spwn.sh/packages/authentication"
	authgh "spwn.sh/packages/authentication/gh"
	authgoogle "spwn.sh/packages/authentication/google"
	authmcp "spwn.sh/packages/authentication/mcp"
	"spwn.sh/packages/platform"
)

// Cmd is the auth command group. The bare invocation renders the
// Credential dashboard: auto-validates, shows every supported
// Method for every provider, surfaces the exact command to set or
// Refresh each one. Unknown positional args fail loudly so that
// Retired verbs (`status`, `check`, `token`) don't silently succeed
// With misleading output: users get an "unknown command" and see
// The real command list in help.
var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage credentials: dashboard, login, use, logout, disable",
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

	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "Save an API key for this provider")
	logoutCmd.Flags().StringVar(&logoutMethod, "method", "", "Scope logout to a single method (oauth | api_key)")
	defaultCmd.Flags().Bool("clear", false, "Remove the default preference (revert to auto-resolve)")
}

var providers = []authentication.Provider{authentication.ProviderAnthropic, authentication.ProviderOpenAI}

var methods = []authentication.Method{authentication.MethodOAuth, authentication.MethodAPIKey}

func runStatus() error {
	w := os.Stderr
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	type providerBlock struct {
		title  string
		rows   []dashboardRow
		active string
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

	renderCLISection(w)

	renderMCPSection(w)

	def := authentication.DefaultProvider()
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

type dashboardRow struct {
	glyph       string
	method      string
	detail      string
	hintCommand string
}

func renderDashboardRow(w io.Writer, row dashboardRow) {
	const methodCol = 14
	methodPadded := ui.PadVisible(ui.Strong(row.method), methodCol)
	line := fmt.Sprintf("    %s %s %s", row.glyph, methodPadded, ui.Faint(row.detail))
	if row.hintCommand != "" {
		line += "  " + ui.Faint("·") + "  " + ui.Cyan(row.hintCommand)
	}
	fmt.Fprintln(w, line)
}

func renderProviderBlock(ctx context.Context, p authentication.Provider) struct {
	title  string
	rows   []dashboardRow
	active string
} {
	result := struct {
		title  string
		rows   []dashboardRow
		active string
	}{title: providerTitle(p)}

	methods := authentication.MethodCatalog(p)
	if len(methods) == 0 {
		return result
	}

	detected := authentication.DetectMethods(p)
	disabled := authentication.IsProviderDisabled(p)
	byMethod := map[authentication.Method]*authentication.Credential{}
	for _, c := range detected {
		if _, seen := byMethod[c.Method()]; !seen {
			byMethod[c.Method()] = c
		}
	}

	type valResult struct {
		method authentication.Method
		cred   *authentication.Credential
		status *authentication.ProviderStatus
	}
	results := make(chan valResult, len(methods))
	var wg sync.WaitGroup
	for _, m := range methods {
		cred := byMethod[authentication.Method(m)]
		if cred == nil {
			continue
		}
		wg.Add(1)
		go func(m authentication.HintMethod, cred *authentication.Credential) {
			defer wg.Done()
			results <- valResult{
				method: authentication.Method(m),
				cred:   cred,
				status: authentication.ValidateWithCache(ctx, cred, 5*time.Minute),
			}
		}(m, cred)
	}
	go func() { wg.Wait(); close(results) }()
	validations := map[authentication.Method]valResult{}
	for r := range results {
		validations[r.method] = r
	}

	var validCreds []*authentication.Credential
	for _, m := range methods {
		method := authentication.Method(m)
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
			row.hintCommand = authentication.NotSetHint(p, m)
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
				row.hintCommand = authentication.RejectedHint(p, cred)
			}
		}
		result.rows = append(result.rows, row)
	}

	if len(validCreds) >= 2 && !disabled {
		pref := authentication.ActiveMethod(p)
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

func providerTitle(p authentication.Provider) string {
	switch p {
	case authentication.ProviderAnthropic:
		return "Anthropic"
	case authentication.ProviderOpenAI:
		return "OpenAI"
	}
	return string(p)
}

func credSourceDetail(cred *authentication.Credential) string {
	if cred == nil {
		return ""
	}
	return cred.Source
}

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
		var m authentication.Method
		if len(args) == 2 {
			m, err = parseMethod(args[1])
			if err != nil {
				return err
			}
		}
		if err := authentication.SetActiveMethod(p, m); err != nil {
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
run the upstream CLI login first, then re-run this command; spwn will
detect the new credential and record it:

  claude login   # then: spwn auth login anthropic
  codex login    # then: spwn auth login openai

For hosted MCP providers (Notion, ...), the command runs the OAuth dance
itself in a one-shot helper container: click the link it prints,
approve in your browser, the token persists for every world:

  spwn auth login notion`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.EqualFold(args[0], "github") {
			return runGitHubLogin(cmd)
		}
		if strings.EqualFold(args[0], "google") {
			return runGoogleLogin(cmd)
		}
		if mp, ok := authmcp.Lookup(args[0]); ok {
			return runMCPLogin(cmd, mp)
		}
		p, err := parseProvider(args[0])
		if err != nil {
			return fmt.Errorf("%w (or one of the MCP providers: %s; or one of the CLI tools: github, google)", err, strings.Join(authmcp.Names(), ", "))
		}
		s := ui.New()
		s.Blank()

		if loginAPIKey != "" {
			if err := saveAPIKey(p, strings.TrimSpace(loginAPIKey)); err != nil {
				return s.FailHint("Save failed", err,
					"Check permissions on "+abbreviate(platform.BaseDir()))
			}
			_ = authentication.SyncCredentials()
			s.Done(fmt.Sprintf("%s API key saved", p), "run `spwn auth` to confirm")
			s.Blank()
			return nil
		}

		_ = authentication.SyncCredentials()
		return runStatus()
	},
}

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
		fmt.Sprintf("ready: `gh ...` and `gh-mcp ...` inside any world will use these (cache: %s)", abbreviate(authgh.CacheDir())))
	s.Blank()
	return nil
}

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
		fmt.Sprintf("ready: `notion-mcp ...` inside any world will use these tokens (cache: %s)", abbreviate(authmcp.CacheDir())))
	s.Blank()
	return nil
}

func saveAPIKey(p authentication.Provider, key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	switch p {
	case authentication.ProviderAnthropic:
		return authentication.SaveToken(key)
	case authentication.ProviderOpenAI:
		return errors.New("OpenAI API key persistence not yet supported; export OPENAI_API_KEY in your shell for now")
	}
	return fmt.Errorf("unsupported provider %q", p)
}

func runGoogleLogin(cmd *cobra.Command) error {
	s := ui.New()
	s.Blank()

	c, err := authgoogle.LoadClient()
	if err != nil {
		return s.FailHint("read client.json failed", err,
			"Check ~/.spwn/credentials/google/ permissions")
	}
	if c == nil {
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
		fmt.Sprintf("ready: `gmail-mcp ...` and `gcal-mcp ...` inside any world will use these (cache: %s)", abbreviate(authgoogle.CacheDir())))
	s.Blank()
	return nil
}

var logoutMethod string

var logoutCmd = &cobra.Command{
	Use:   "logout <provider>",
	Short: "Clear stored credentials for a provider",
	Long: `Remove every stored credential for a provider: cache file,
macOS keychain entry, runtime-CLI auth files. Does NOT unset env vars
(the shell owns those); any active env vars are surfaced so you know
to unset them manually.

  spwn auth logout anthropic
  spwn auth logout openai
  spwn auth logout anthropic --method api_key   # spare keychain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		if strings.EqualFold(args[0], "google") {
			s := ui.New()
			s.Blank()
			removed := false
			for _, path := range []string{authgoogle.TokensPath(), authgoogle.ClientPath()} {
				err := os.Remove(path)
				if err == nil {
					removed = true
				} else if !errors.Is(err, os.ErrNotExist) {
					return s.FailHint("google logout failed", err,
						"Inspect "+abbreviate(authgoogle.CacheDir())+" by hand if this persists")
				}
			}
			if !removed {
				s.Info("google", "already logged out")
				s.Blank()
				return nil
			}
			s.Done("Removed", "google credentials ("+abbreviate(authgoogle.CacheDir())+")")
			s.Blank()
			return nil
		}
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
			return fmt.Errorf("%w (or one of the MCP providers: %s; or one of the CLI tools: github, google)", err, strings.Join(authmcp.Names(), ", "))
		}
		opts := authentication.LogoutOpts{}
		if logoutMethod != "" {
			opts.Method, err = parseMethod(logoutMethod)
			if err != nil {
				return err
			}
		}
		s := ui.New()
		s.Blank()

		res := authentication.LogoutProvider(p, opts)

		if len(res.Cleared) == 0 && len(res.Remaining) == 0 && !res.HasErrors() {
			s.Info(string(p), "already logged out")
			s.Blank()
			return nil
		}
		for _, item := range res.Cleared {
			s.Done("Removed", item)
		}
		for _, item := range res.Remaining {
			s.Warn("Still set (shell env)", item+"  (run `unset "+strings.TrimPrefix(item, "env:")+"` to clear)")
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
		if err := authentication.DisableProvider(p); err != nil {
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
		if err := authentication.EnableProvider(p); err != nil {
			return err
		}
		s := ui.New()
		s.Blank()
		s.Success(fmt.Sprintf("%s enabled", p))
		s.Blank()
		return runStatus()
	},
}

var defaultCmd = &cobra.Command{
	Use:   "default [provider]",
	Short: "Pick which provider spwn prefers when multiple are authenticated",
	Long: `Set a soft preference for which provider's runtime spwn picks
when you're logged into more than one and no runtime is pinned at the
project or agent level.

This is the durable answer to the "multiple providers authenticated
and no runtime pinned" error. Set it once and spwn will quietly
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
			if err := authentication.SetDefaultProvider(""); err != nil {
				return err
			}
			s.Blank()
			s.Success("default provider cleared")
			s.Blank()
			return nil
		}

		if len(args) == 0 {
			current := authentication.DefaultProvider()
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
		if authentication.IsProviderDisabled(p) {
			return s.FailHint("Default refused",
				fmt.Errorf("%s is currently disabled", p),
				fmt.Sprintf("Run `spwn auth enable %s` first, or pick a different provider", p))
		}
		if err := authentication.SetDefaultProvider(p); err != nil {
			return err
		}
		s.Blank()
		s.Success(fmt.Sprintf("default provider set to %s", p))
		s.Blank()
		return nil
	},
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
			{Title: "Inspect", Commands: []ui.HelpEntry{
				{Name: "(bare)", Desc: "Live credential dashboard, validated against each provider API"},
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
				{Name: "anthropic", Desc: "Claude: OAuth (subscription) or API key"},
				{Name: "openai", Desc: "Codex: OAuth (ChatGPT Plus) or API key (env-only today)"},
			}},
		},
		"spwn auth [command]",
		"",
	)
}

func parseProvider(raw string) (authentication.Provider, error) {
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

func parseMethod(raw string) (authentication.Method, error) {
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
