package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

// token cache file name inside SPWN_HOME
const tokenFile = ".auth-token"

func tokenPath() string {
	return filepath.Join(foundation.BaseDir(), tokenFile)
}

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
		s.Blank()

		// Detect each provider
		anthropic := detectAnthropic()
		openai := detectProvider("OPENAI_API_KEY", "OpenAI")
		google := detectProvider("GOOGLE_API_KEY", "Google")

		// Table
		t := ui.NewTable(ui.ModeNormal, "PROVIDER", "STATUS", "SOURCE")
		t.AddRow("Anthropic", statusText(anthropic.ok), anthropic.source)
		t.AddRow("OpenAI", statusText(openai.ok), openai.source)
		t.AddRow("Google", statusText(google.ok), google.source)
		t.Render()

		// Token cache info
		s.Blank()
		if info, err := os.Stat(tokenPath()); err == nil {
			age := time.Since(info.ModTime())
			s.Info("Cached token:", fmt.Sprintf("%s (%s ago)", abbreviate(tokenPath()), formatAge(age)))
		} else {
			s.Info("Cached token:", ui.Faint("none"))
		}

		s.Blank()
		if !anthropic.ok {
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

		// 1. Try macOS Keychain
		s.Start("Checking Keychain...")
		token := extractKeychainToken()
		if token != "" {
			// Cache it
			os.MkdirAll(foundation.BaseDir(), 0755)
			os.WriteFile(tokenPath(), []byte(token), 0600)
			s.Done("Found subscription", "cached to "+abbreviate(tokenPath()))
			s.Blank()
			s.Success("Authenticated.")
			s.Blank()
			return nil
		}
		s.Done("Keychain", ui.Faint("no Claude Code credentials found"))

		// 2. Check env vars
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			s.Done("Environment", "ANTHROPIC_API_KEY is set")
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
		os.MkdirAll(foundation.BaseDir(), 0755)
		if err := os.WriteFile(tokenPath(), []byte(input), 0600); err != nil {
			return s.FailHint("Save failed", err, "Check permissions on "+abbreviate(foundation.BaseDir()))
		}

		s.Blank()
		s.Done("Saved "+kind, abbreviate(tokenPath()))
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

		path := tokenPath()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			s.Info("No cached token", "nothing to clear")
			s.Blank()
			return nil
		}

		if err := os.Remove(path); err != nil {
			return s.FailHint("Clear failed", err, "Try removing manually: rm "+path)
		}

		s.Done("Cleared cached token", abbreviate(path))

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

		os.MkdirAll(foundation.BaseDir(), 0755)
		if err := os.WriteFile(tokenPath(), []byte(input), 0600); err != nil {
			return s.FailHint("Save failed", err, "Check permissions on "+abbreviate(foundation.BaseDir()))
		}

		kind := "token"
		if strings.HasPrefix(input, "sk-ant-") {
			kind = "API key"
		}

		s.Done("Saved "+kind, abbreviate(tokenPath()))
		s.Blank()
		return nil
	},
}

// --- Helpers ---

type providerStatus struct {
	ok     bool
	source string
}

func detectAnthropic() providerStatus {
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return providerStatus{true, "ANTHROPIC_API_KEY"}
	}
	if os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != "" {
		return providerStatus{true, "CLAUDE_CODE_OAUTH_TOKEN"}
	}
	if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
		return providerStatus{true, "ANTHROPIC_AUTH_TOKEN"}
	}

	// Check cached token
	if data, err := os.ReadFile(tokenPath()); err == nil && strings.TrimSpace(string(data)) != "" {
		return providerStatus{true, "cached token"}
	}

	// Try Keychain (non-blocking, quick check)
	if token := extractKeychainToken(); token != "" {
		return providerStatus{true, "Keychain"}
	}

	return providerStatus{false, "\u2014"}
}

func detectProvider(envKey, name string) providerStatus {
	if os.Getenv(envKey) != "" {
		return providerStatus{true, envKey}
	}
	return providerStatus{false, "\u2014"}
}

func statusText(ok bool) string {
	if ok {
		return ui.Green("\u2713") + " configured"
	}
	return ui.Faint("\u25CB") + " not set"
}

func extractKeychainToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return ""
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return ""
	}
	return creds.ClaudeAiOauth.AccessToken
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
