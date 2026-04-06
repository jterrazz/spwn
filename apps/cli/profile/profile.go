package profile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v3"
)

// ── flags ───────────────────────────────────────────────────────────────────

var (
	editFlag   bool
	limitFlag  int
	allFlag    bool
	setFlag    string
)

// ── help ────────────────────────────────────────────────────────────────────

var defaultProfileHelp func(*cobra.Command, []string)

func init() {
	defaultProfileHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(profileHelp)
}

func profileHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "profile" {
		if defaultProfileHelp != nil {
			defaultProfileHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ profile")+" "+ui.Faint("— view and edit an agent's character sheet"),
		[]ui.HelpGroup{
			{Title: "Core", Commands: []ui.HelpEntry{
				{Name: "purpose", Desc: "Why the agent exists"},
				{Name: "traits", Desc: "Core principles and character"},
				{Name: "persona", Desc: "Personality and role"},
			}},
			{Title: "Capabilities", Commands: []ui.HelpEntry{
				{Name: "skills", Desc: "Learned capabilities"},
				{Name: "playbooks", Desc: "Step-by-step procedures"},
			}},
			{Title: "Memory", Commands: []ui.HelpEntry{
				{Name: "knowledge", Desc: "Facts and context"},
				{Name: "journal", Desc: "Session and deployment history"},
			}},
			{Title: "Config", Commands: []ui.HelpEntry{
				{Name: "edit", Desc: "Open profile.yaml in $EDITOR"},
				{Name: "role", Desc: "View/change role"},
				{Name: "engine", Desc: "View/change runtime engine"},
			}},
		},
		"spwn profile <name>          Show full character sheet\n    spwn profile <name> [aspect]",
		"Use \"spwn profile <name> <aspect> --help\" for more information.",
	)
}

// ── Cmd ─────────────────────────────────────────────────────────────────────

// Cmd is the profile command group.
var Cmd = &cobra.Command{
	Use:   "profile <name> [subcommand]",
	Short: "View and edit agent profiles",
	Long:  `View and edit an agent's character sheet — identity, skills, memory, and configuration.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if len(args) == 1 {
			return showCharacterSheet(cmd, name)
		}

		aspect := args[1]
		rest := args[2:]

		switch aspect {
		// Core identity files
		case "purpose":
			return showFile(cmd, name, filepath.Join("core", "purpose.md"), "purpose")
		case "traits":
			return showFile(cmd, name, filepath.Join("core", "traits.md"), "traits")
		case "persona":
			return showFile(cmd, name, filepath.Join("core", "persona.md"), "persona")

		// Directory listings
		case "skills":
			return listDir(cmd, name, "skills", "skills")
		case "playbooks":
			return listDir(cmd, name, "playbooks", "playbooks")
		case "knowledge":
			return listDir(cmd, name, "knowledge", "knowledge")

		// Journal (merged with sessions)
		case "journal":
			return showJournal(cmd, name)

		// Config
		case "role":
			return showOrSetRole(cmd, name)
		case "engine":
			return showOrSetEngine(cmd, name)
		case "edit":
			return editProfile(cmd, name)

		default:
			_ = rest
			return fmt.Errorf("unknown profile aspect %q — run \"spwn profile --help\" for available aspects", aspect)
		}
	},
}

func init() {
	Cmd.Flags().BoolVar(&editFlag, "edit", false, "Open file in $EDITOR")
	Cmd.Flags().IntVar(&limitFlag, "limit", 10, "Number of journal entries to show")
	Cmd.Flags().BoolVar(&allFlag, "all", false, "Show all journal entries")
	Cmd.Flags().StringVar(&setFlag, "set", "", "Set value (for role/engine)")
}

// ── helpers ─────────────────────────────────────────────────────────────────

func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}

func agentExists(name string) bool {
	dir := agentDomain.AgentDir(name)
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

func agentNotFoundError(name string) error {
	return fmt.Errorf("agent %q not found — create one with \"spwn agent new %s\"", name, name)
}

// ── ProfileYAML ─────────────────────────────────────────────────────────────

// ProfileYAML represents the profile.yaml manifest.
type ProfileYAML struct {
	Role    string        `yaml:"role,omitempty"`
	Runtime RuntimeConfig `yaml:"runtime,omitempty"`
}

// RuntimeConfig describes the agent's runtime engine.
type RuntimeConfig struct {
	Engine   string `yaml:"engine,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

func loadProfileYAML(name string) (*ProfileYAML, error) {
	path := filepath.Join(agentDomain.AgentDir(name), "profile.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultProfile(), nil
		}
		return nil, err
	}
	var p ProfileYAML
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	applyProfileDefaults(&p)
	return &p, nil
}

func saveProfileYAML(name string, p *ProfileYAML) error {
	path := filepath.Join(agentDomain.AgentDir(name), "profile.yaml")
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func defaultProfile() *ProfileYAML {
	p := &ProfileYAML{}
	applyProfileDefaults(p)
	return p
}

func applyProfileDefaults(p *ProfileYAML) {
	if p.Role == "" {
		p.Role = "worker"
	}
	if p.Runtime.Engine == "" {
		p.Runtime.Engine = "claude-code"
	}
	if p.Runtime.Provider == "" {
		p.Runtime.Provider = "anthropic"
	}
	if p.Runtime.Model == "" {
		p.Runtime.Model = "sonnet"
	}
}

// readFirstLine reads the first non-empty line of a file.
func readFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

// countFiles returns the number of .md files in a directory.
func countFiles(dir string) (int, int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	count := 0
	var totalSize int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		count++
		if fi, err := e.Info(); err == nil {
			totalSize += fi.Size()
		}
	}
	return count, totalSize
}

// fileNames returns the base names (without extension) of .md files.
func fileNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	return names
}

// formatSize formats a byte count into a human-readable string.
func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}

// timeAgo returns a human-friendly relative time string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(d.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// ── character sheet ─────────────────────────────────────────────────────────

func showCharacterSheet(cmd *cobra.Command, name string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	w := cmd.ErrOrStderr()
	mindPath := agentDomain.AgentDir(name)
	profile, _ := loadProfileYAML(name)

	// Gather stats
	created := "unknown"
	if fi, err := os.Stat(mindPath); err == nil {
		created = timeAgo(fi.ModTime())
	}

	journalEntries, _ := agentDomain.ListJournal(mindPath, 0)
	journalCount := len(journalEntries)

	// Core identity
	purposeLine := readFirstLine(filepath.Join(mindPath, "core", "purpose.md"))
	if purposeLine == "" {
		purposeLine = "not set"
	}
	traitsLine := readFirstLine(filepath.Join(mindPath, "core", "traits.md"))
	if traitsLine == "" {
		traitsLine = "not set"
	}
	personaLine := readFirstLine(filepath.Join(mindPath, "core", "persona.md"))
	if personaLine == "" {
		personaLine = "not set"
	}

	// Capabilities
	skillCount, _ := countFiles(filepath.Join(mindPath, "skills"))
	skillNames := fileNames(filepath.Join(mindPath, "skills"))
	playbookCount, _ := countFiles(filepath.Join(mindPath, "playbooks"))

	// Memory
	knowledgeCount, knowledgeSize := countFiles(filepath.Join(mindPath, "knowledge"))

	// Engine string
	engineStr := fmt.Sprintf("%s · %s · %s", profile.Runtime.Engine, profile.Runtime.Provider, profile.Runtime.Model)

	// ── Render ──

	boxWidth := 56
	inner := boxWidth - 4 // inside │ ... │

	fmt.Fprintln(w)
	// Top border
	fmt.Fprintf(w, "  ╭─ %s %s╮\n", ui.Strong(name), strings.Repeat("─", max(1, inner-len(name)-2)))
	fmt.Fprintf(w, "  │%s│\n", strings.Repeat(" ", inner+2))

	// General
	roleLabel := profile.Role
	if roleLabel == "" {
		roleLabel = "(none)"
	}
	printSheetRow(w, inner, "Role", roleLabel)
	printSheetRow(w, inner, "Engine", engineStr)
	printSheetRow(w, inner, "Created", created)

	// Core identity section
	fmt.Fprintf(w, "  │%s│\n", strings.Repeat(" ", inner+2))
	fmt.Fprintf(w, "  │  ── Core %s│\n", strings.Repeat("─", max(1, inner-10)))
	printSheetRow(w, inner, "Purpose", purposeLine)
	printSheetRow(w, inner, "Traits", traitsLine)
	printSheetRow(w, inner, "Persona", personaLine)

	// Capabilities section
	fmt.Fprintf(w, "  │%s│\n", strings.Repeat(" ", inner+2))
	fmt.Fprintf(w, "  │  ── Capabilities %s│\n", strings.Repeat("─", max(1, inner-19)))
	skillsValue := fmt.Sprintf("%d files", skillCount)
	if skillCount > 0 && len(skillNames) <= 5 {
		skillsValue = fmt.Sprintf("%d files (%s)", skillCount, strings.Join(skillNames, ", "))
	}
	printSheetRow(w, inner, "Skills", skillsValue)
	printSheetRow(w, inner, "Playbooks", fmt.Sprintf("%d files", playbookCount))

	// Memory section
	fmt.Fprintf(w, "  │%s│\n", strings.Repeat(" ", inner+2))
	fmt.Fprintf(w, "  │  ── Memory %s│\n", strings.Repeat("─", max(1, inner-12)))
	printSheetRow(w, inner, "Knowledge", fmt.Sprintf("%d files · %s", knowledgeCount, formatSize(knowledgeSize)))
	printSheetRow(w, inner, "Journal", fmt.Sprintf("%d entries", journalCount))

	// Bottom
	fmt.Fprintf(w, "  │%s│\n", strings.Repeat(" ", inner+2))
	fmt.Fprintf(w, "  ╰%s╯\n", strings.Repeat("─", inner+2))
	fmt.Fprintln(w)

	return nil
}

func printSheetRow(w io.Writer, inner int, label, value string) {
	line := fmt.Sprintf("  %-10s %s", label, value)
	padding := inner + 2 - len(line)
	if padding < 1 {
		padding = 1
	}
	fmt.Fprintf(w, "  │%s%s│\n", line, strings.Repeat(" ", padding))
}

// titleCase capitalises the first rune of a string (replaces deprecated strings.Title).
func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ── file viewer ─────────────────────────────────────────────────────────────

func showFile(cmd *cobra.Command, name, relPath, aspect string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	fullPath := filepath.Join(agentDomain.AgentDir(name), relPath)

	if editFlag {
		return openInEditor(fullPath, aspect)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			s := newStepper(cmd)
			s.Blank()
			s.Info(titleCase(aspect)+":", "Not set yet.")
			s.Log("Create with: spwn profile %s %s --edit", name, aspect)
			s.Blank()
			return nil
		}
		return err
	}

	fmt.Fprint(cmd.ErrOrStderr(), string(data))
	return nil
}

func openInEditor(path, aspect string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Create with template if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		template := fmt.Sprintf("# %s\n\n", titleCase(aspect))
		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			return err
		}
	}

	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ── directory listing ───────────────────────────────────────────────────────

func listDir(cmd *cobra.Command, name, relDir, label string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	dir := filepath.Join(agentDomain.AgentDir(name), relDir)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		s := newStepper(cmd)
		s.Blank()
		s.Info(titleCase(label)+":", fmt.Sprintf("No %s yet.", label))
		s.Blank()
		return nil
	}

	w := cmd.ErrOrStderr()
	t := ui.NewTable(ui.ModeNormal, "FILE", "DESCRIPTION", "SIZE")

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fpath := filepath.Join(dir, e.Name())
		desc := readFirstLine(fpath)
		if desc == "" {
			desc = "(no description)"
		}
		size := ""
		if fi, err := e.Info(); err == nil {
			size = formatSize(fi.Size())
		}
		t.AddRow(
			strings.TrimSuffix(e.Name(), ".md"),
			desc,
			size,
		)
	}

	_ = w
	t.Render()
	return nil
}

// ── journal ─────────────────────────────────────────────────────────────────

func showJournal(cmd *cobra.Command, name string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	mindPath := agentDomain.AgentDir(name)
	limit := limitFlag
	if allFlag {
		limit = 0
	}

	entries, err := agentDomain.ListJournal(mindPath, limit)
	if err != nil {
		return fmt.Errorf("cannot read journal: %w", err)
	}

	if len(entries) == 0 {
		s := newStepper(cmd)
		s.Blank()
		s.Info("Journal:", "No sessions yet.")
		s.Log("Spawn the agent into a world to create journal entries.")
		s.Blank()
		return nil
	}

	t := ui.NewTable(ui.ModeNormal, "DATE", "WORLD", "EXIT", "DURATION")
	for _, e := range entries {
		t.AddRow(
			e.CreatedAt.Format("2006-01-02"),
			e.WorldID,
			fmt.Sprintf("%d", e.ExitCode),
			ui.FormatDuration(e.Duration),
		)
	}
	t.Render()
	return nil
}

// ── role ────────────────────────────────────────────────────────────────────

func showOrSetRole(cmd *cobra.Command, name string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	if setFlag != "" {
		p, err := loadProfileYAML(name)
		if err != nil {
			return err
		}
		p.Role = setFlag
		if err := saveProfileYAML(name, p); err != nil {
			return err
		}
		s := newStepper(cmd)
		s.Blank()
		s.Done("Role updated", setFlag)
		s.Blank()
		return nil
	}

	p, err := loadProfileYAML(name)
	if err != nil {
		return err
	}

	s := newStepper(cmd)
	s.Blank()
	roleLabel := p.Role
	if roleLabel == "" {
		roleLabel = "(none)"
	}
	s.Info("Role:", roleLabel)
	s.Blank()
	return nil
}

// ── engine ──────────────────────────────────────────────────────────────────

func showOrSetEngine(cmd *cobra.Command, name string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	if setFlag != "" {
		p, err := loadProfileYAML(name)
		if err != nil {
			return err
		}
		p.Runtime.Engine = setFlag
		if err := saveProfileYAML(name, p); err != nil {
			return err
		}
		s := newStepper(cmd)
		s.Blank()
		s.Done("Engine updated", setFlag)
		s.Blank()
		return nil
	}

	p, err := loadProfileYAML(name)
	if err != nil {
		return err
	}

	s := newStepper(cmd)
	s.Blank()
	s.Info("Engine:", p.Runtime.Engine)
	s.Info("Provider:", p.Runtime.Provider)
	s.Info("Model:", p.Runtime.Model)
	s.Blank()
	return nil
}

// ── edit ─────────────────────────────────────────────────────────────────────

func editProfile(cmd *cobra.Command, name string) error {
	if !agentExists(name) {
		return agentNotFoundError(name)
	}

	path := filepath.Join(agentDomain.AgentDir(name), "profile.yaml")

	// Create with defaults if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		p := defaultProfile()
		if err := saveProfileYAML(name, p); err != nil {
			return err
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

