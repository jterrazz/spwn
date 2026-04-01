package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

// --- helpers -----------------------------------------------------------------

// abbreviatePath replaces the user's home directory with ~.
func abbreviatePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// padRight pads a string with spaces to at least width visible characters.
func padRight(s string, width int) string {
	visible := visibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// visibleLen returns the number of visible characters in a string,
// ignoring ANSI escape sequences. It counts UTF-8 runes.
func visibleLen(s string) int {
	clean := stripAnsi(s)
	return utf8.RuneCountInString(clean)
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until terminator letter
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++ // skip terminator
			}
			i = j
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

// repeatStr repeats a string n times (for single-char strings like "─").
func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// w is a shortcut for writing to stderr.
var w = os.Stderr

func pr(format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}

// --- status command ----------------------------------------------------------

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the full status of your spwn environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		const boxWidth = 64 // outer width of the header box

		// ── Gather data ─────────────────────────────────────────────────

		org, _ := universe.LoadOrg()
		orgName := ""
		if org != nil && org.Name != "" {
			orgName = org.Name
		}

		baseDir := foundation.BaseDir()

		// Auth & skills
		authLabel := "not configured"
		authToken := ""
		if data, err := os.ReadFile(filepath.Join(baseDir, ".auth-token")); err == nil {
			authToken = strings.TrimSpace(string(data))
		}
		if authToken != "" {
			authLabel = "subscription"
		} else if os.Getenv("ANTHROPIC_API_KEY") != "" {
			authLabel = "API key"
		}

		skillCount := 0
		if entries, err := os.ReadDir(foundation.SkillsDir()); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					skillCount++
				}
			}
		}

		// Physics from default manifest
		m, err := universe.LoadManifest("default")
		if err != nil {
			// Use built-in defaults if no config
			universe.ApplyDefaults(&m)
		}
		cpu := fmt.Sprintf("%d cpu", m.Physics.Constants.CPU)
		mem := m.Physics.Constants.Memory
		timeout := m.Physics.Constants.Timeout

		// Worlds
		var worlds []universe.World
		arc, arcErr := universe.NewArchitectFromEnv()
		if arcErr == nil {
			worlds, _ = arc.List(context.Background())
		}

		// Agents
		agentList, _ := agentDomain.ListAgents()

		// Build map: agent name → world (to identify limbo agents)
		worldMap := make(map[string]*universe.World)
		for i := range worlds {
			w := &worlds[i]
			if w.Agent != "" {
				worldMap[w.Agent] = w
			}
			for _, a := range w.Agents {
				worldMap[a.Name] = w
			}
		}

		// ── Header box ──────────────────────────────────────────────────

		pr("\n")

		// Top border
		pr("  %s%s%s\n", "\u256d", repeatStr("\u2500", boxWidth-2), "\u256e")

		// Line 1: branding + version
		brandLine := "\u2b21  s p w n"
		versionStr := "v" + Version
		innerWidth := boxWidth - 6 // visible chars between │ margins: │ __ content __ │
		brandPad := innerWidth - visibleLen(brandLine) - len(versionStr)
		if brandPad < 1 {
			brandPad = 1
		}
		pr("  \u2502  %s%s%s  \u2502\n", brandLine, strings.Repeat(" ", brandPad), ui.Faint(versionStr))

		// Line 2: org name + home
		orgLine := ""
		if orgName != "" {
			orgLine = orgName + " \u00b7 "
		}
		orgLine += abbreviatePath(baseDir)
		pr("  \u2502  %s  \u2502\n", padRight(orgLine, innerWidth))

		// Line 3: auth + skills
		authLine := ""
		if authLabel == "subscription" || authLabel == "API key" {
			authLine = "\u2713 " + authLabel
		} else {
			authLine = authLabel
		}
		authLine += fmt.Sprintf(" \u00b7 %d skills", skillCount)
		pr("  \u2502  %s  \u2502\n", padRight(authLine, innerWidth))

		// Bottom border
		pr("  %s%s%s\n", "\u2570", repeatStr("\u2500", boxWidth-2), "\u256f")

	// ── Architect section ────────────────────────────────────────────

	pr("\n")
	pr("  \u2726 %s \u00b7 %s\n", ui.Strong("Architect"), ui.Faint("offline"))
		pr("  \u2502  %s   %s\n", ui.Faint("channels"), "\u2014")
		pr("  \u2502  %s       %s\n", ui.Faint("sync"), "\u2014")

		// ── Universe section ────────────────────────────────────────────

		pr("\n")
		pr("  \u25c9 %s \u00b7 %s \u00b7 %s \u00b7 %s\n",
			ui.Strong("Universe"), ui.Faint(cpu), ui.Faint(mem), ui.Faint(timeout))

		// Active worlds
		activeWorlds := make([]universe.World, 0)
		for _, ww := range worlds {
			if ww.Status == universe.StatusRunning || ww.Status == universe.StatusIdle || ww.Status == universe.StatusCreating {
				activeWorlds = append(activeWorlds, ww)
			}
		}

		if len(activeWorlds) > 0 {
			for _, ww := range activeWorlds {
				pr("  \u2502\n")
				renderWorldBubble(ww, agentList)
			}
		}

		// ── Limbo section ────────────────────────────────────────────

		// Limbo agents: those not attached to any world
		var limbo []agentDomain.Info
		for _, a := range agentList {
			if _, attached := worldMap[a.Name]; !attached {
				limbo = append(limbo, a)
			}
		}

		pr("  \u2502\n")
		pr("  \u2570\u2500\u2500\u25cc %s\n", ui.Strong("Limbo"))

		if len(limbo) > 0 {
			for _, a := range limbo {
				layers := fmt.Sprintf("%d/6", agentDomain.LayerCount(&a))
				pr("     \u2502\n")
				pr("     \u2502  \u25cc %s    %s\n", padRight(a.Name, 10), ui.Faint(layers))
			}
			pr("     \u2502\n")
			pr("     \u2570%s\n", repeatStr("\u2500", 20))
		} else {
			pr("     \u2502\n")
			pr("     \u2570%s\n", repeatStr("\u2500", 20))
		}

		pr("\n")
		return nil
	},
}

// renderWorldBubble draws a world bubble with agents inside.
func renderWorldBubble(ww universe.World, allAgents []agentDomain.Info) {
	// Collect agents in this world
	type agentEntry struct {
		name   string
		tier   string
		layers string
		status string
	}

	var agents []agentEntry

	// Multi-agent worlds
	if len(ww.Agents) > 0 {
		for _, ar := range ww.Agents {
			tier := ar.Tier
			if tier == "" {
				tier = "citizen"
			}
			layers := "?/6"
			for _, ai := range allAgents {
				if ai.Name == ar.Name {
					layers = fmt.Sprintf("%d/6", agentDomain.LayerCount(&ai))
					break
				}
			}
			statusIcon := "\u25cc idle"
			if ar.Status == universe.StatusRunning {
				statusIcon = "\u25cf active"
			}
			agents = append(agents, agentEntry{
				name:   ar.Name,
				tier:   tier,
				layers: layers,
				status: statusIcon,
			})
		}
	} else if ww.Agent != "" {
		// Legacy single-agent
		tier := "citizen"
		layers := "?/6"
		for _, ai := range allAgents {
			if ai.Name == ww.Agent {
				layers = fmt.Sprintf("%d/6", agentDomain.LayerCount(&ai))
				break
			}
		}
		statusIcon := "\u25cc idle"
		if ww.Status == universe.StatusRunning {
			statusIcon = "\u25cf active"
		}
		agents = append(agents, agentEntry{
			name:   ww.Agent,
			tier:   tier,
			layers: layers,
			status: statusIcon,
		})
	}

	// Calculate bubble width based on content
	minWidth := 47
	// Check agent lines: icon(2) + space + name(10) + gap(3) + tier(10) + gap(3) + layers(3) + gap(3) + status(8) + trail(4)
	for _, a := range agents {
		lineLen := 4 + utf8.RuneCountInString(a.name) + 3 + utf8.RuneCountInString(a.tier) + 3 + utf8.RuneCountInString(a.layers) + 3 + utf8.RuneCountInString(a.status) + 6
		if lineLen > minWidth {
			minWidth = lineLen
		}
	}

	// Check header line width
	headerContent := ww.ID + " " + ww.Config
	headerMin := utf8.RuneCountInString(headerContent) + 12
	if headerMin > minWidth {
		minWidth = headerMin
	}

	// Workspace line
	wsAbbrev := ""
	if ww.Workspace != "" {
		wsAbbrev = abbreviatePath(ww.Workspace)
	}
	if ww.Workspace != "" {
		wsLineLen := 14 + len(wsAbbrev) + 4
		if wsLineLen > minWidth {
			minWidth = wsLineLen
		}
	}

	bubbleInner := minWidth

	// Elements from manifest
	elements := ""
	if len(ww.Manifest.Elements) > 0 {
		// Show @pack names (not expanded)
		elems := make([]string, len(ww.Manifest.Elements))
		copy(elems, ww.Manifest.Elements)
		elements = strings.Join(elems, " ")
	}

	// Uptime
	uptime := "\u2014"
	if !ww.CreatedAt.IsZero() {
		dur := time.Since(ww.CreatedAt)
		uptime = ui.FormatDuration(dur)
	}

	// ── Draw bubble ─────────────────────────────────────────────────

	// Top line: ╭─ w-default-28373 ──────────────── default ─╮
	leftLabel := " " + ww.ID + " "
	rightLabel := " " + ww.Config + " "
	fillLen := bubbleInner - len(leftLabel) - len(rightLabel)
	if fillLen < 1 {
		fillLen = 1
	}
	pr("  \u2502  \u256d\u2500%s%s%s\u2500\u256e\n", leftLabel, repeatStr("\u2500", fillLen), rightLabel)

	// Empty line
	pr("  \u2502  \u2502%s\u2502\n", strings.Repeat(" ", bubbleInner))

	// Agent lines
	for _, a := range agents {
		icon := "\u25cf" // ●
		if a.tier == "governor" {
			icon = "\u2605" // ★
		}
		agentLine := fmt.Sprintf("   %s %s   %s   %s   %s",
			icon,
			padRight(a.name, 10),
			padRight(a.tier, 10),
			a.layers,
			a.status,
		)
		pr("  \u2502  \u2502%s\u2502\n", padRight(agentLine, bubbleInner))
	}

	// Empty line
	pr("  \u2502  \u2502%s\u2502\n", strings.Repeat(" ", bubbleInner))

	// Workspace
	if ww.Workspace != "" {
		wsLine := fmt.Sprintf("   workspace  %s", wsAbbrev)
		pr("  \u2502  \u2502%s\u2502\n", padRight(wsLine, bubbleInner))
	}

	// Elements
	if elements != "" {
		elemLine := fmt.Sprintf("   elements   %s", elements)
		pr("  \u2502  \u2502%s\u2502\n", padRight(elemLine, bubbleInner))
	}

	// Uptime
	uptimeLine := fmt.Sprintf("   uptime     %s", uptime)
	pr("  \u2502  \u2502%s\u2502\n", padRight(uptimeLine, bubbleInner))

	// Bottom border
	pr("  \u2502  \u2570%s\u256f\n", repeatStr("\u2500", bubbleInner))
}
