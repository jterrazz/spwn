package observatory

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	agentpkg "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/architect"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/state"

	"gopkg.in/yaml.v3"
)

const webVersionCheckInterval = 1 * time.Hour

// Server serves the Observatory HTTP API.
type Server struct {
	state *state.Store
	arch  *architect.Architect // nil = read-only mode
	addr  string
	srv   *http.Server
}

// New creates an Observatory server. arch may be nil for read-only mode.
func New(s *state.Store, arch *architect.Architect, addr string) *Server {
	return &Server{state: s, arch: arch, addr: addr}
}

// cors wraps a handler with CORS headers.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// jsonError writes an error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// jsonOK writes a JSON success response.
func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// requireArch checks that the architect is available (non-read-only mode).
func (s *Server) requireArch(w http.ResponseWriter) bool {
	if s.arch == nil {
		jsonError(w, "server is in read-only mode (no architect configured)", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// Start begins serving the API.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// --- READ endpoints ---
	mux.HandleFunc("GET /api/health", cors(s.handleHealth))
	mux.HandleFunc("GET /api/status", cors(s.handleStatus))
	mux.HandleFunc("GET /api/version", cors(s.handleVersion))
	mux.HandleFunc("GET /api/universes", cors(s.handleListUniverses))
	mux.HandleFunc("GET /api/agents", cors(s.handleListAgents))
	mux.HandleFunc("GET /api/agents/{name}", cors(s.handleGetAgent))
	mux.HandleFunc("GET /api/agents/{name}/journal", cors(s.handleGetAgentJournal))
	mux.HandleFunc("GET /api/agents/{name}/mind", cors(s.handleGetAgentMind))
	mux.HandleFunc("GET /api/agents/{name}/files/{path...}", cors(s.handleGetAgentFile))
	mux.HandleFunc("GET /api/worlds/{id}/logs", cors(s.handleWorldLogs))

	// --- WRITE endpoints ---
	mux.HandleFunc("POST /api/worlds", cors(s.handleCreateWorld))
	mux.HandleFunc("DELETE /api/worlds/{id}", cors(s.handleDestroyWorld))
	mux.HandleFunc("POST /api/worlds/{id}/snapshot", cors(s.handleSnapshot))
	mux.HandleFunc("POST /api/agents", cors(s.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{name}", cors(s.handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{name}/dream", cors(s.handleDream))
	mux.HandleFunc("POST /api/agents/{name}/sleep", cors(s.handleSleep))
	mux.HandleFunc("POST /api/agents/{name}/fork", cors(s.handleFork))
	mux.HandleFunc("POST /api/worlds/{id}/talk", cors(s.handleTalk))
	mux.HandleFunc("POST /api/agents/{name}/export", cors(s.handleExport))
	mux.HandleFunc("PUT /api/agents/{name}/identity", cors(s.handleUpdateIdentity))

	// --- Architect endpoints ---
	mux.HandleFunc("GET /api/architect/status", cors(s.handleArchitectStatus))
	mux.HandleFunc("POST /api/architect/start", cors(s.handleArchitectStart))
	mux.HandleFunc("POST /api/architect/stop", cors(s.handleArchitectStop))
	mux.HandleFunc("POST /api/architect/talk", cors(s.handleArchitectTalk))
	mux.HandleFunc("GET /api/architect/directives", cors(s.handleArchitectDirectivesGet))
	mux.HandleFunc("POST /api/architect/directives", cors(s.handleArchitectDirectivesUpdate))

	// --- Blueprint endpoints ---
	mux.HandleFunc("GET /api/blueprint", cors(s.handleBlueprintList))
	mux.HandleFunc("GET /api/blueprint/{path...}", cors(s.handleBlueprintRead))
	mux.HandleFunc("PUT /api/blueprint/{path...}", cors(s.handleBlueprintWrite))

	// --- Root redirect + CORS ---
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"name":    "spwn observatory API",
				"version": foundation.Version,
				"docs":    "/api/health",
				"dashboard": "http://localhost:3000",
			})
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("OPTIONS /", cors(func(w http.ResponseWriter, r *http.Request) {}))

	s.srv = &http.Server{Addr: s.addr, Handler: mux}
	fmt.Printf("Observatory API listening on %s\n", s.addr)
	return s.srv.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// ============================================================
// READ handlers
// ============================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	worlds, _ := s.state.List()
	agents, _ := agentpkg.ListAgents()

	vi := foundation.GetVersionInfo(webVersionCheckInterval)

	status := map[string]interface{}{
		"worlds":          len(worlds),
		"agents":          len(agents),
		"architect":       s.arch != nil,
		"version":         vi.Current,
		"latestVersion":   vi.Latest,
		"updateAvailable": vi.UpdateAvailable,
	}
	jsonOK(w, status)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	vi := foundation.GetVersionInfo(webVersionCheckInterval)
	jsonOK(w, vi)
}

func (s *Server) handleListUniverses(w http.ResponseWriter, r *http.Request) {
	universes, err := s.state.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, universes)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := agentpkg.ListAgents()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, agents)
}

// profileYAML represents the profile.yaml manifest for an agent.
type profileYAML struct {
	Tier    string `yaml:"tier,omitempty" json:"tier,omitempty"`
	Runtime struct {
		Engine   string `yaml:"engine,omitempty" json:"engine,omitempty"`
		Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
		Model    string `yaml:"model,omitempty" json:"model,omitempty"`
	} `yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

// agentProfileResponse is the full profile sent to the frontend.
type agentProfileResponse struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Tier      string            `json:"tier"`
	Engine    string            `json:"engine"`
	Provider  string            `json:"provider"`
	Purpose   string            `json:"purpose"`
	Persona   string            `json:"persona"`
	Traits    []string          `json:"traits"`
	Skills    []string          `json:"skills"`
	Playbooks []string          `json:"playbooks"`
	Knowledge []string          `json:"knowledge"`
	Journal   []journalEntry    `json:"journal"`
	Bonds     []bondEntry       `json:"bonds"`
	Layers    map[string][]string `json:"layers"`
}

type journalEntry struct {
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

type bondEntry struct {
	Agent        string `json:"agent"`
	Relationship string `json:"relationship"`
}

// readFirstLineContent reads the first non-empty, non-heading line of a file.
func readFirstLineContent(path string) string {
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

// listMdFiles returns base names (without .md extension) of markdown files in a directory.
func listMdFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			result = append(result, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return result
}

// parseTraits reads a traits.md file and returns individual traits as a slice.
func parseTraits(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var traits []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip leading list markers (-, *, •)
		line = strings.TrimLeft(line, "-*• ")
		line = strings.TrimSpace(line)
		if line != "" {
			traits = append(traits, line)
		}
	}
	return traits
}

// parseBonds reads bonds.md and returns structured bond entries.
func parseBonds(path string) []bondEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var bonds []bondEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimLeft(line, "-*• ")
		line = strings.TrimSpace(line)
		// Try "agent: relationship" or "agent — relationship" format
		var agent, rel string
		if idx := strings.Index(line, ":"); idx > 0 {
			agent = strings.TrimSpace(line[:idx])
			rel = strings.TrimSpace(line[idx+1:])
		} else if idx := strings.Index(line, "—"); idx > 0 {
			agent = strings.TrimSpace(line[:idx])
			rel = strings.TrimSpace(line[idx+len("—"):])
		} else if idx := strings.Index(line, "-"); idx > 0 {
			agent = strings.TrimSpace(line[:idx])
			rel = strings.TrimSpace(line[idx+1:])
		} else {
			agent = line
			rel = "connected"
		}
		if agent != "" {
			bonds = append(bonds, bondEntry{Agent: agent, Relationship: rel})
		}
	}
	return bonds
}

// parseJournalFiles reads journal directory and returns structured entries.
func parseJournalFiles(dir string) []journalEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var journal []journalEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		date := strings.TrimSuffix(e.Name(), ".md")
		// Try to read first content line as summary
		summary := readFirstLineContent(filepath.Join(dir, e.Name()))
		journal = append(journal, journalEntry{Date: date, Summary: summary})
	}
	return journal
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	info, err := agentpkg.InspectAgent(name)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	mindPath := info.Path

	// Load profile.yaml for tier/engine/provider
	tier := "citizen"
	engine := "claude-code"
	provider := "anthropic"
	profilePath := filepath.Join(mindPath, "profile.yaml")
	if data, err := os.ReadFile(profilePath); err == nil {
		var p profileYAML
		if err := yaml.Unmarshal(data, &p); err == nil {
			if p.Tier != "" {
				tier = p.Tier
			}
			if p.Runtime.Engine != "" {
				engine = p.Runtime.Engine
			}
			if p.Runtime.Provider != "" {
				provider = p.Runtime.Provider
			}
		}
	}

	// Read identity files
	purpose := readFirstLineContent(filepath.Join(mindPath, "identity", "purpose.md"))
	persona := readFirstLineContent(filepath.Join(mindPath, "identity", "persona.md"))
	traits := parseTraits(filepath.Join(mindPath, "identity", "traits.md"))
	if traits == nil {
		traits = []string{}
	}

	// List capability files
	skills := listMdFiles(filepath.Join(mindPath, "skills"))
	if skills == nil {
		skills = []string{}
	}
	playbooks := listMdFiles(filepath.Join(mindPath, "memory", "playbooks"))
	if playbooks == nil {
		playbooks = []string{}
	}
	knowledge := listMdFiles(filepath.Join(mindPath, "memory", "knowledge"))
	if knowledge == nil {
		knowledge = []string{}
	}

	// Journal entries
	journal := parseJournalFiles(filepath.Join(mindPath, "memory", "journal"))
	if journal == nil {
		// Try legacy path
		journal = parseJournalFiles(filepath.Join(mindPath, "journal"))
	}
	if journal == nil {
		journal = []journalEntry{}
	}

	// Bonds
	bonds := parseBonds(filepath.Join(mindPath, "bonds.md"))
	if bonds == nil {
		bonds = []bondEntry{}
	}

	resp := agentProfileResponse{
		Name:      info.Name,
		Path:      info.Path,
		Tier:      tier,
		Engine:    engine,
		Provider:  provider,
		Purpose:   purpose,
		Persona:   persona,
		Traits:    traits,
		Skills:    skills,
		Playbooks: playbooks,
		Knowledge: knowledge,
		Journal:   journal,
		Bonds:     bonds,
		Layers:    info.Layers,
	}

	jsonOK(w, resp)
}

func (s *Server) handleGetAgentJournal(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	mindPath := agentpkg.AgentDir(name)
	entries, err := agentpkg.ListJournal(mindPath, 100)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, entries)
}

func (s *Server) handleWorldLogs(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}

	reader, err := s.arch.Logs(r.Context(), worldID, follow, tail)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	defer reader.Close()

	if follow {
		// SSE streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			jsonError(w, "streaming not supported", 500)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if line != "" {
						fmt.Fprintf(w, "data: %s\n\n", line)
					}
				}
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		io.Copy(w, reader)
	}
}

// ============================================================
// WRITE handlers
// ============================================================

func (s *Server) handleCreateWorld(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	var body struct {
		ConfigName string `json:"config"`
		AgentName  string `json:"agent"`
		Workspace  string `json:"workspace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	// Load manifest with defaults (same as CLI does)
	cfgName := body.ConfigName
	if cfgName == "" {
		cfgName = "default"
	}
	m, err := manifest.Load(cfgName)
	if err != nil {
		jsonError(w, "config not found: "+err.Error(), 400)
		return
	}
	manifest.ApplyDefaults(&m)

	result, err := s.arch.Spawn(r.Context(), architect.SpawnOpts{
		ConfigName: cfgName,
		AgentName:  body.AgentName,
		Workspace:  body.Workspace,
		Manifest:   m,
	})
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, result)
}

func (s *Server) handleDestroyWorld(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	world, err := s.arch.Destroy(r.Context(), worldID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, world)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) // optional body

	tag, err := s.arch.Snapshot(r.Context(), worldID, body.Name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"tag": tag})
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "name is required", 400)
		return
	}

	path, err := agentpkg.InitMind(body.Name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"name": body.Name, "path": path})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	if err := agentpkg.DeleteAgent(name); err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonOK(w, map[string]string{"deleted": name})
}

func (s *Server) handleDream(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	result, err := agentpkg.Dream(name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, result)
}

func (s *Server) handleSleep(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	result, err := agentpkg.Sleep(name)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, result)
}

func (s *Server) handleFork(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	var body struct {
		Target string   `json:"target"`
		Layers []string `json:"layers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Target == "" {
		jsonError(w, "target is required", 400)
		return
	}

	result, err := agentpkg.Fork(name, body.Target, body.Layers)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, result)
}

func (s *Server) handleTalk(w http.ResponseWriter, r *http.Request) {
	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		jsonError(w, "message is required", 400)
		return
	}

	// Find the world to get agent name
	universes, err := s.state.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	var agentName string
	for _, u := range universes {
		if u.ID == worldID {
			agentName = u.Agent
			break
		}
	}
	if agentName == "" {
		jsonError(w, "world not found or has no agent", 404)
		return
	}

	// Execute spwn agent talk with streaming JSON output
	cmd := exec.CommandContext(r.Context(), "spwn", "agent", "talk", agentName, body.Message,
		"--output-format", "stream-json", "--verbose")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		jsonError(w, "failed to create stdout pipe: "+err.Error(), 500)
		return
	}
	cmd.Stderr = nil // ignore stderr

	if err := cmd.Start(); err != nil {
		jsonError(w, "failed to start agent talk: "+err.Error(), 500)
		return
	}

	// Stream as SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large tool results
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	cmd.Wait()
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	tarPath, err := agentpkg.ExportMind(name, "", nil)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", name))
	http.ServeFile(w, r, tarPath)
}

func (s *Server) handleArchitectStatus(w http.ResponseWriter, r *http.Request) {
	// Check if the spwn-architect Docker container is actually running
	status := "stopped"
	var containerID interface{} = nil
	var uptime interface{} = nil

	cmd := exec.CommandContext(r.Context(), "docker", "inspect", "--format", "{{.State.Status}}|{{.Id}}|{{.State.StartedAt}}", "spwn-architect")
	if output, err := cmd.Output(); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 3)
		if len(parts) >= 1 && parts[0] == "running" {
			status = "running"
			if len(parts) >= 2 {
				containerID = parts[1]
			}
			if len(parts) >= 3 {
				if started, err := time.Parse(time.RFC3339Nano, parts[2]); err == nil {
					dur := time.Since(started).Truncate(time.Second)
					uptime = dur.String()
				}
			}
		}
	}

	// Build KPIs — always included regardless of Docker status
	worlds, _ := s.state.List()
	agents, _ := agentpkg.ListAgents()

	// Count directives from file
	tasksPending := 0
	tasksCompleted := 0
	directivesPath := s.architectDirectivesPath()
	if data, err := os.ReadFile(directivesPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
				tasksCompleted++
			} else if strings.HasPrefix(trimmed, "- [ ]") {
				tasksPending++
			}
		}
	}

	jsonOK(w, map[string]interface{}{
		"status":      status,
		"containerId": containerID,
		"uptime":      uptime,
		"kpis": map[string]interface{}{
			"worlds":         len(worlds),
			"agents":         len(agents),
			"tasksPending":   tasksPending,
			"tasksCompleted": tasksCompleted,
		},
	})
}

// architectDirectivesPath returns the path to the architect's directives file.
func (s *Server) architectDirectivesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	spwnHome := os.Getenv("SPWN_HOME")
	if spwnHome == "" {
		spwnHome = filepath.Join(home, ".spwn")
	}
	return filepath.Join(spwnHome, "architect", "directives.md")
}

// handleArchitectDirectivesGet returns the raw content of the architect TODO file.
func (s *Server) handleArchitectDirectivesGet(w http.ResponseWriter, r *http.Request) {
	directivesPath := s.architectDirectivesPath()
	if directivesPath == "" {
		jsonError(w, "could not determine directives path", 500)
		return
	}

	data, err := os.ReadFile(directivesPath)
	if err != nil {
		// Return empty template if file doesn't exist
		defaultContent := "# Architect Directives\n\n## In Progress\n\n## Backlog\n\n## Completed\n"
		jsonOK(w, map[string]string{"content": defaultContent})
		return
	}

	jsonOK(w, map[string]string{"content": string(data)})
}

// handleArchitectDirectivesUpdate writes new content to the architect TODO file.
func (s *Server) handleArchitectDirectivesUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	directivesPath := s.architectDirectivesPath()
	if directivesPath == "" {
		jsonError(w, "could not determine directives path", 500)
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(directivesPath), 0755); err != nil {
		jsonError(w, "failed to create directory: "+err.Error(), 500)
		return
	}

	if err := os.WriteFile(directivesPath, []byte(body.Content), 0644); err != nil {
		jsonError(w, "failed to write directives file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleArchitectStart(w http.ResponseWriter, r *http.Request) {
	// Use `spwn architect start` which handles container creation + startup
	cmd := exec.CommandContext(r.Context(), "spwn", "architect", "start")
	if output, err := cmd.CombinedOutput(); err != nil {
		jsonError(w, fmt.Sprintf("failed to start architect container: %s (%s)", strings.TrimSpace(string(output)), err), 500)
		return
	}
	jsonOK(w, map[string]string{"status": "running"})
}

func (s *Server) handleArchitectStop(w http.ResponseWriter, r *http.Request) {
	// Try to stop the spwn-architect container
	cmd := exec.CommandContext(r.Context(), "docker", "stop", "spwn-architect")
	if output, err := cmd.CombinedOutput(); err != nil {
		jsonError(w, fmt.Sprintf("failed to stop architect container: %s (%s)", strings.TrimSpace(string(output)), err), 500)
		return
	}
	jsonOK(w, map[string]string{"status": "stopped"})
}

// DirectiveAction represents a parsed directive action from the architect's response.
type DirectiveAction struct {
	Type        string `json:"type"`                  // "add", "done", "update"
	Title       string `json:"title"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

// BlueprintUpdate represents a parsed blueprint update marker from the architect's response.
type BlueprintUpdate struct {
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// parseDirectiveAction extracts a TODO action marker from the architect's response.
// It looks for [DIRECTIVE_ADD], [DIRECTIVE_DONE], or [DIRECTIVE_UPDATE] at the start of lines.
func parseDirectiveAction(text string) *DirectiveAction {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		var actionType string
		var prefix string
		switch {
		case strings.HasPrefix(trimmed, "[DIRECTIVE_ADD]"):
			actionType = "add"
			prefix = "[DIRECTIVE_ADD]"
		case strings.HasPrefix(trimmed, "[DIRECTIVE_DONE]"):
			actionType = "done"
			prefix = "[DIRECTIVE_DONE]"
		case strings.HasPrefix(trimmed, "[DIRECTIVE_UPDATE]"):
			actionType = "update"
			prefix = "[DIRECTIVE_UPDATE]"
		default:
			continue
		}

		title := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		action := &DirectiveAction{
			Type:  actionType,
			Title: title,
		}

		// Look at subsequent lines for Priority: and description
		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				break
			}
			if strings.HasPrefix(next, "Priority:") {
				action.Priority = strings.TrimSpace(strings.TrimPrefix(next, "Priority:"))
			} else if strings.HasPrefix(next, "Completed:") {
				action.Description = strings.TrimSpace(strings.TrimPrefix(next, "Completed:"))
			} else if strings.HasPrefix(next, "Progress:") {
				action.Description = strings.TrimSpace(strings.TrimPrefix(next, "Progress:"))
			} else if action.Description == "" {
				action.Description = next
			}
		}

		return action
	}
	return nil
}

// parseBlueprintUpdate extracts a [BLUEPRINT_UPDATE] marker from the architect's response.
func parseBlueprintUpdate(text string) *BlueprintUpdate {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "[BLUEPRINT_UPDATE]") {
			continue
		}

		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "[BLUEPRINT_UPDATE]"))
		update := &BlueprintUpdate{
			Path: path,
		}

		// Look at the next line for a description
		if i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && !strings.HasPrefix(next, "[") {
				update.Description = next
			}
		}

		return update
	}
	return nil
}

func (s *Server) handleArchitectTalk(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}
	if body.Message == "" {
		jsonError(w, "message is required", 400)
		return
	}

	// Check if architect is running, auto-start if needed
	containerName := "spwn-architect"
	checkCmd := exec.CommandContext(r.Context(), "docker", "inspect", "--format", "{{.State.Status}}", containerName)
	checkOut, checkErr := checkCmd.Output()
	isRunning := checkErr == nil && strings.TrimSpace(string(checkOut)) == "running"

	if !isRunning {
		// Try to start the architect container
		startCmd := exec.CommandContext(r.Context(), "docker", "start", containerName)
		if startErr := startCmd.Run(); startErr != nil {
			// Container doesn't exist, try spwn architect start
			spwnCmd := exec.CommandContext(r.Context(), "spwn", "architect", "start")
			if spwnErr := spwnCmd.Run(); spwnErr != nil {
				jsonError(w, "failed to start architect: "+spwnErr.Error(), 500)
				return
			}
		}
		// Wait for container to be ready
		time.Sleep(3 * time.Second)
	}

	// Docker exec into the architect container running Claude Code
	dockerArgs := []string{
		"exec", "-u", "architect", "-w", "/world",
		"-e", "SPWN_HOME=/spwn-data",
	}
	// Pass auth tokens
	for _, key := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_AUTH_TOKEN"} {
		if val := os.Getenv(key); val != "" {
			dockerArgs = append(dockerArgs, "-e", key+"="+val)
		}
	}
	// Also try cached token
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") == "" {
		tokenPath := filepath.Join(foundation.BaseDir(), ".auth-token")
		if data, err := os.ReadFile(tokenPath); err == nil {
			token := strings.TrimSpace(string(data))
			if token != "" {
				if strings.HasPrefix(token, "sk-ant-") {
					dockerArgs = append(dockerArgs, "-e", "ANTHROPIC_API_KEY="+token)
				} else {
					dockerArgs = append(dockerArgs, "-e", "CLAUDE_CODE_OAUTH_TOKEN="+token)
				}
			}
		}
	}
	dockerArgs = append(dockerArgs, containerName,
		"claude", "--dangerously-skip-permissions",
		"-p", body.Message,
		"--output-format", "stream-json", "--verbose",
		"--append-system-prompt",
		"You are the Architect. Read /world/ARCHITECT.md for your identity. "+
			"IMPORTANT: When a user asks you to do something, you MUST include a [DIRECTIVE_ADD] marker in your response. "+
			"Format: [DIRECTIVE_ADD] Short directive title\nPriority: high|medium|low\nBrief description. "+
			"Also update /world/directives.md with the new directive. "+
			"When completing a directive use [DIRECTIVE_DONE] Short directive title. "+
			"Read /world/skills/ for detailed guides. "+
			"BLUEPRINT: You maintain /blueprint/ as the single source of truth. "+
			"When updating blueprint files, include [BLUEPRINT_UPDATE] path/to/file.md in your response. "+
			"Every conversation should result in blueprint updates.",
	)

	cmd := exec.CommandContext(r.Context(), "docker", dockerArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		jsonError(w, "failed to create stdout pipe: "+err.Error(), 500)
		return
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		jsonError(w, "failed to start architect talk: "+err.Error(), 500)
		return
	}

	// Stream as SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	cmd.Wait()
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// blueprintBasePath returns the path to the blueprint directory.
func (s *Server) blueprintBasePath() string {
	return filepath.Join(foundation.BaseDir(), "blueprint")
}

// handleBlueprintList returns all files in the blueprint directory.
func (s *Server) handleBlueprintList(w http.ResponseWriter, r *http.Request) {
	basePath := s.blueprintBasePath()

	type fileEntry struct {
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		Modified string `json:"modified"`
	}

	var files []fileEntry

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		files = append(files, fileEntry{
			Path:     relPath,
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339),
		})
		return nil
	})

	if err != nil {
		// If directory doesn't exist, return empty list
		if os.IsNotExist(err) {
			jsonOK(w, map[string]interface{}{"files": []fileEntry{}})
			return
		}
		jsonError(w, "failed to list blueprint files: "+err.Error(), 500)
		return
	}

	if files == nil {
		files = []fileEntry{}
	}

	jsonOK(w, map[string]interface{}{"files": files})
}

// handleBlueprintRead returns the content of a specific blueprint file.
func (s *Server) handleBlueprintRead(w http.ResponseWriter, r *http.Request) {
	relPath := r.PathValue("path")
	if relPath == "" {
		jsonError(w, "file path is required", 400)
		return
	}

	// Prevent directory traversal
	if strings.Contains(relPath, "..") {
		jsonError(w, "invalid path", 400)
		return
	}

	absPath := filepath.Join(s.blueprintBasePath(), relPath)

	// Ensure the resolved path is still under the blueprint directory
	cleanPath := filepath.Clean(absPath)
	cleanBase := filepath.Clean(s.blueprintBasePath())
	if !strings.HasPrefix(cleanPath, cleanBase) {
		jsonError(w, "path outside blueprint directory", 400)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		jsonError(w, "file not found: "+err.Error(), 404)
		return
	}

	jsonOK(w, map[string]string{"path": relPath, "content": string(data)})
}

// handleBlueprintWrite writes content to a specific blueprint file.
func (s *Server) handleBlueprintWrite(w http.ResponseWriter, r *http.Request) {
	relPath := r.PathValue("path")
	if relPath == "" {
		jsonError(w, "file path is required", 400)
		return
	}

	// Prevent directory traversal
	if strings.Contains(relPath, "..") {
		jsonError(w, "invalid path", 400)
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	absPath := filepath.Join(s.blueprintBasePath(), relPath)

	// Ensure the resolved path is still under the blueprint directory
	cleanPath := filepath.Clean(absPath)
	cleanBase := filepath.Clean(s.blueprintBasePath())
	if !strings.HasPrefix(cleanPath, cleanBase) {
		jsonError(w, "path outside blueprint directory", 400)
		return
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		jsonError(w, "failed to create directory: "+err.Error(), 500)
		return
	}

	if err := os.WriteFile(absPath, []byte(body.Content), 0644); err != nil {
		jsonError(w, "failed to write file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "path": relPath})
}

// handleGetAgentMind returns the mind tree (layers → files) for an agent.
func (s *Server) handleGetAgentMind(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	info, err := agentpkg.InspectAgent(name)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	jsonOK(w, info.Layers)
}

// handleUpdateIdentity updates a single identity field for an agent.
func (s *Server) handleUpdateIdentity(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}

	var body struct {
		Field   string `json:"field"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	// Validate field name to prevent directory traversal
	allowed := map[string]bool{"purpose": true, "persona": true, "traits": true, "bonds": true}
	if !allowed[body.Field] {
		jsonError(w, "invalid field: must be one of purpose, persona, traits, bonds", 400)
		return
	}

	info, err := agentpkg.InspectAgent(name)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	identityDir := filepath.Join(info.Path, "identity")
	if err := os.MkdirAll(identityDir, 0755); err != nil {
		jsonError(w, "failed to create identity dir: "+err.Error(), 500)
		return
	}

	filePath := filepath.Join(identityDir, body.Field+".md")

	// Write content with a heading
	heading := strings.ToUpper(body.Field[:1]) + body.Field[1:]
	content := fmt.Sprintf("# %s\n\n%s\n", heading, body.Content)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		jsonError(w, "failed to write file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "field": body.Field})
}

// handleGetAgentFile returns the content of a specific file within the agent's mind directory.
// The path is validated to prevent directory traversal.
func (s *Server) handleGetAgentFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	filePath := r.PathValue("path")
	if name == "" || filePath == "" {
		jsonError(w, "agent name and file path are required", 400)
		return
	}

	// Prevent directory traversal
	if strings.Contains(filePath, "..") {
		jsonError(w, "invalid path", 400)
		return
	}

	info, err := agentpkg.InspectAgent(name)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	absPath := filepath.Join(info.Path, filePath)

	// Ensure the resolved path is still under the agent's mind directory
	cleanPath := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(info.Path)) {
		jsonError(w, "path outside agent directory", 400)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		jsonError(w, "file not found: "+err.Error(), 404)
		return
	}

	jsonOK(w, map[string]string{"path": filePath, "content": string(data)})
}
