package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/world"
	"spwn.sh/packages/paths"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

// --- helpers -----------------------------------------------------------------

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

func padRight(s string, width int) string {
	visible := visibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func visibleLen(s string) int {
	return utf8.RuneCountInString(stripAnsi(s))
}

func stripAnsi(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

var w = os.Stderr

func pr(format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}

// ruleWidth is the total visible width of section rule lines.
const ruleWidth = 56

// rule renders a section header like: ── Label ──── right ──
func rule(label, right string) string {
	left := "\u2500\u2500 " + label + " "
	rightPart := ""
	if right != "" {
		rightPart = " " + right + " \u2500\u2500"
	}
	fillLen := ruleWidth - visibleLen(left) - visibleLen(rightPart)
	if fillLen < 1 {
		fillLen = 1
	}
	return ui.Faint(left + repeatStr("\u2500", fillLen) + rightPart)
}

// --- status command ----------------------------------------------------------

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the full status of your spwn environment",
	RunE: func(cmd *cobra.Command, args []string) error {

		// ── Gather data ─────────────────────────────────────────────

		baseDir := paths.BaseDir()

		// Auth
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

		// Skills
		skillCount := 0
		if entries, err := os.ReadDir(paths.SkillsDir()); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					skillCount++
				}
			}
		}

		// Worlds
		var worlds []world.World
		arc, arcErr := world.NewArchitectFromEnv()
		if arcErr == nil {
			worlds, _ = arc.List(context.Background())
		}

		// Agents
		agentList, _ := agent.ListAgents()

		// Agent → world mapping
		worldMap := make(map[string]*world.World)
		for i := range worlds {
			ww := &worlds[i]
			if ww.Agent != "" {
				worldMap[ww.Agent] = ww
			}
			for _, a := range ww.Agents {
				worldMap[a.Name] = ww
			}
		}

		// Active worlds
		var activeWorlds []world.World
		for _, ww := range worlds {
			if ww.Status == world.StatusRunning || ww.Status == world.StatusIdle || ww.Status == world.StatusCreating {
				activeWorlds = append(activeWorlds, ww)
			}
		}

		// Idle agents (not attached to any active world)
		var idleAgents []agent.Info
		for _, a := range agentList {
			if _, attached := worldMap[a.Name]; !attached {
				idleAgents = append(idleAgents, a)
			}
		}

		// ── Render ──────────────────────────────────────────────────

		pr("\n")

		// ── Header ──────────────────────────────────────────────────

		pr("  %s %s\n", ui.Strong("spwn"), ui.Faint("v"+Version))

		infoparts := []string{ui.Faint(abbreviatePath(baseDir))}
		if authLabel == "subscription" || authLabel == "API key" {
			infoparts = append(infoparts, ui.Green("\u2713")+" "+ui.Faint(authLabel))
		} else {
			infoparts = append(infoparts, ui.Faint(authLabel))
		}
		infoparts = append(infoparts, ui.Faint(fmt.Sprintf("%d skills", skillCount)))
		pr("  %s\n", strings.Join(infoparts, ui.Faint(" \u00b7 ")))

		// ── World sections ──────────────────────────────────────────

		pr("\n")

		if len(activeWorlds) > 0 {
			for _, ww := range activeWorlds {
				renderWorldSection(ww)
			}
		} else {
			pr("  %s\n", rule("Worlds", "none"))
			pr("     %s\n", ui.Faint("spwn up --agent <name> -w ."))
			pr("\n")
		}

		// ── Idle agents ─────────────────────────────────────────────

		if len(idleAgents) > 0 {
			pr("  %s\n", rule("Idle", fmt.Sprintf("%d agent%s", len(idleAgents), plural(len(idleAgents)))))
			for _, a := range idleAgents {
				pr("     %s %s\n", ui.Faint("\u25cb"), a.Name)
			}
			pr("\n")
		}

		// ── Architect ───────────────────────────────────────────────

		pr("  %s\n", rule("Architect", "offline"))

		pr("\n")
		return nil
	},
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderWorldSection draws a world as a rule-header section with agents listed below.
func renderWorldSection(ww world.World) {
	// ── Section header rule ─────────────────────────────────────

	// Left: world ID + optional config
	label := ww.ID
	if ww.Config != "" && ww.Config != "default" {
		label += " \u00b7 " + ww.Config
	}

	// Right: status + uptime
	statusText := string(ww.Status)
	uptime := ""
	if !ww.CreatedAt.IsZero() {
		uptime = ui.FormatDuration(time.Since(ww.CreatedAt))
	}
	rightText := statusText
	if uptime != "" {
		rightText += " \u00b7 " + uptime
	}

	pr("  %s\n", rule(label, rightText))

	// ── Detail lines (indented) ─────────────────────────────────

	indent := "     "

	// Workspaces
	wsPaths := collectWorkspacePaths(ww)
	if len(wsPaths) > 0 {
		pr("%s%s\n", indent, strings.Join(wsPaths, ", "))
	}

	// Tools
	if len(ww.Manifest.Tools) > 0 {
		pr("%s%s\n", indent, ui.Faint(strings.Join(ww.Manifest.Tools, " ")))
	}

	// Agents
	agents := collectAgents(ww)
	if len(agents) > 0 {
		pr("\n")
		for _, a := range agents {
			dot := ui.Faint("\u25cb") // ○
			if a.status == world.StatusRunning {
				dot = ui.Green("\u25cf") // ●
			}
			rolePart := padRight(a.role, 8)
			statusPart := string(a.status)
			pr("%s%s %s %s %s\n", indent, dot, padRight(a.name, 12), ui.Faint(rolePart), ui.Faint(statusPart))
		}
	}

	pr("\n")
}

type agentInfo struct {
	name   string
	role   string
	status world.Status
}

func collectAgents(ww world.World) []agentInfo {
	var agents []agentInfo
	if len(ww.Agents) > 0 {
		for _, ar := range ww.Agents {
			role := ar.Role
			if role == "" {
				role = "worker"
			}
			agents = append(agents, agentInfo{
				name:   ar.Name,
				role:   role,
				status: ar.Status,
			})
		}
	} else if ww.Agent != "" {
		agents = append(agents, agentInfo{
			name:   ww.Agent,
			role:   "worker",
			status: ww.Status,
		})
	}
	return agents
}

func collectWorkspacePaths(ww world.World) []string {
	var paths []string
	if len(ww.Workspaces) > 0 {
		for _, ws := range ww.Workspaces {
			paths = append(paths, abbreviatePath(ws.Path))
		}
	}
	return paths
}
