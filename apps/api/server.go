package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"spwn.sh/packages/activity"
	agentpkg "spwn.sh/packages/agent"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/authentication"
	"spwn.sh/packages/container/probe"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/platform"
	projectpkg "spwn.sh/packages/project"
	worldpkg "spwn.sh/packages/world"
	"spwn.sh/packages/world/manifest"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtimestate"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/upgrade"
)

const webVersionCheckInterval = 1 * time.Hour

// Server serves the HTTP API.
type Server struct {
	rstate *runtimestate.Store
	arch   *architect.Architect
	addr   string
	srv    *http.Server

	spawnArchitectFn ArchitectSpawnFunc

	architectMu         sync.Mutex
	architectSpawnState *architectSpawn
}

// SetSpawnArchitect wires the implementation of the architect daemon
// spawn function. Must be called before /api/architect/start is hit
// for the first time. The cli/cmd wiring is responsible for passing
// architect.StartDaemonWithOpts here.
func (s *Server) SetSpawnArchitect(fn ArchitectSpawnFunc) {
	s.spawnArchitectFn = fn
}

// New creates an API server. arch may be nil for read-only mode.
func New(s *runtimestate.Store, arch *architect.Architect, addr string) *Server {
	return &Server{rstate: s, arch: arch, addr: addr}
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) requireArch(w http.ResponseWriter) bool {
	if s.arch == nil {
		jsonError(w, "server is in read-only mode (no architect configured)", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// Handler returns the production HTTP handler. Tests use this same
// router so route registration cannot drift from Start().
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return mux
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", cors(s.handleHealth))
	mux.HandleFunc("GET /api/status", cors(s.handleStatus))
	mux.HandleFunc("GET /api/version", cors(s.handleVersion))
	mux.HandleFunc("GET /api/system/docker", cors(s.handleSystemDocker))
	mux.HandleFunc("GET /api/system/onboarding", cors(s.handleSystemOnboarding))
	mux.HandleFunc("POST /api/system/onboarding/complete", cors(s.handleSystemOnboardingComplete))
	mux.HandleFunc("GET /api/examples", cors(s.handleListExamples))
	mux.HandleFunc("GET /api/examples/{slug}", cors(s.handleGetExample))
	mux.HandleFunc("POST /api/examples/{slug}/install", cors(s.handleInstallExample))
	mux.HandleFunc("GET /api/worlds", cors(s.handleListWorlds))
	mux.HandleFunc("GET /api/agents", cors(s.handleListAgents))
	mux.HandleFunc("GET /api/agents/{name}", cors(s.handleGetAgent))
	mux.HandleFunc("GET /api/agents/{name}/journal", cors(s.handleGetAgentJournal))
	mux.HandleFunc("GET /api/agents/{name}/mind", cors(s.handleGetAgentMind))
	mux.HandleFunc("GET /api/agents/{name}/files/{path...}", cors(s.handleGetAgentFile))

	mux.HandleFunc("POST /api/worlds", cors(s.handleCreateWorld))
	mux.HandleFunc("PATCH /api/worlds/{id}", cors(s.handleRenameWorld))
	mux.HandleFunc("DELETE /api/worlds/{id}", cors(s.handleDestroyWorld))
	mux.HandleFunc("POST /api/worlds/{id}/snapshot", cors(s.handleSnapshot))
	mux.HandleFunc("POST /api/agents", cors(s.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{name}", cors(s.handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{name}/dream", cors(s.handleDream))
	mux.HandleFunc("POST /api/agents/{name}/sleep", cors(s.handleSleep))
	mux.HandleFunc("POST /api/agents/{name}/fork", cors(s.handleFork))
	mux.HandleFunc("POST /api/worlds/{id}/agents", cors(s.handleDeployAgent))
	mux.HandleFunc("POST /api/worlds/{id}/talk", cors(s.handleTalk))
	mux.HandleFunc("POST /api/agents/{name}/export", cors(s.handleExport))
	mux.HandleFunc("PUT /api/agents/{name}/identity", cors(s.handleUpdateIdentity))

	mux.HandleFunc("GET /api/teams", cors(s.handleListTeams))
	mux.HandleFunc("POST /api/teams", cors(s.handleCreateTeam))
	mux.HandleFunc("PUT /api/teams/{slug}", cors(s.handleUpdateTeam))
	mux.HandleFunc("DELETE /api/teams/{slug}", cors(s.handleDeleteTeam))

	mux.HandleFunc("GET /api/organizations", cors(s.handleListOrganizations))
	mux.HandleFunc("GET /api/organizations/{slug}", cors(s.handleGetOrganization))
	mux.HandleFunc("POST /api/organizations", cors(s.handleCreateOrganization))
	mux.HandleFunc("PUT /api/organizations/{slug}", cors(s.handleUpdateOrganization))
	mux.HandleFunc("DELETE /api/organizations/{slug}", cors(s.handleDeleteOrganization))

	mux.HandleFunc("GET /api/architect/status", cors(s.handleArchitectStatus))
	mux.HandleFunc("GET /api/architect/history", cors(s.handleArchitectHistory))
	mux.HandleFunc("GET /api/architect/logs", cors(s.handleArchitectLogs))
	mux.HandleFunc("POST /api/architect/start", cors(s.handleArchitectStart))
	mux.HandleFunc("POST /api/architect/stop", cors(s.handleArchitectStop))
	mux.HandleFunc("POST /api/architect/talk", cors(s.handleArchitectTalk))
	mux.HandleFunc("GET /api/architect/stack", cors(s.handleArchitectStackGet))
	mux.HandleFunc("POST /api/architect/stack", cors(s.handleArchitectStackUpdate))

	mux.HandleFunc("GET /api/worlds/{id}/history", cors(s.handleWorldHistory))

	mux.HandleFunc("GET /api/activity", cors(s.handleActivity))

	mux.HandleFunc("GET /api/auth/providers", cors(s.handleAuthProviders))
	mux.HandleFunc("POST /api/auth/check", cors(s.handleAuthCheck))
	mux.HandleFunc("POST /api/auth/configure", cors(s.handleAuthConfigure))
	mux.HandleFunc("POST /api/auth/reset", cors(s.handleAuthReset))
	mux.HandleFunc("POST /api/auth/reconnect", cors(s.handleAuthReconnect))

	mux.HandleFunc("GET /api/worlds/{id}/knowledge", cors(s.handleWorldKnowledgeList))
	mux.HandleFunc("GET /api/worlds/{id}/knowledge/{path...}", cors(s.handleWorldKnowledgeRead))
	mux.HandleFunc("PUT /api/worlds/{id}/knowledge/{path...}", cors(s.handleWorldKnowledgeWrite))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"name":      "spwn spwn API",
				"version":   upgrade.CLIVersion,
				"docs":      "/api/health",
				"dashboard": "http://localhost:3000",
			})
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("OPTIONS /", cors(func(w http.ResponseWriter, r *http.Request) {}))
}

// Start begins serving the API.
func (s *Server) Start() error {
	s.srv = &http.Server{Addr: s.addr, Handler: s.Handler()}
	fmt.Printf("spwn API listening on %s\n", s.addr)
	return s.srv.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleSystemDocker(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, probe.CheckDocker(r.Context()))
}

func (s *Server) handleSystemOnboarding(w http.ResponseWriter, r *http.Request) {
	completed := false
	if _, err := os.Stat(filepath.Join(platform.BaseDir(), ".onboarding-complete")); err == nil {
		completed = true
	}
	worlds, _ := s.rstate.List()
	agents, _ := agentpkg.ListAgents()
	docker := probe.CheckDocker(r.Context())
	hasAuth := false
	for _, c := range authentication.ResolveAll() {
		if c != nil && c.Token != "" {
			hasAuth = true
			break
		}
	}
	jsonOK(w, map[string]interface{}{
		"completed": completed,
		"hasDocker": docker.OK(),
		"hasAuth":   hasAuth,
		"hasWorlds": len(worlds) > 0,
		"hasAgents": len(agents) > 0,
		"docker":    docker,
	})
}

func (s *Server) handleListExamples(w http.ResponseWriter, r *http.Request) {
	list, err := dependency.Gallery()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]interface{}{"examples": list})
}

func (s *Server) handleGetExample(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ex, err := dependency.GalleryEntryBySlug(slug)
	if err != nil {
		if err == dependency.ErrNotFound {
			jsonError(w, "example not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, ex)
}

func (s *Server) handleInstallExample(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	rep, err := dependency.InstallInto(slug)
	if err != nil {
		if err == dependency.ErrNotFound {
			jsonError(w, "example not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, rep)
}

func (s *Server) handleSystemOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(platform.BaseDir(), ".onboarding-complete")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	worlds, _ := s.rstate.List()
	agents, _ := agentpkg.ListAgents()

	vi := upgrade.GetVersionInfo(webVersionCheckInterval)

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
	vi := upgrade.GetVersionInfo(webVersionCheckInterval)
	jsonOK(w, vi)
}

type declaredWorldItem struct {
	Name       string   `json:"name"`
	Agents     []string `json:"agents"`
	Workspaces []string `json:"workspaces"`
	Deps       []string `json:"deps,omitempty"`
	// Status is "running" when any live world carries this config
	// name, "stopped" otherwise. Declared-but-undeployed worlds are
	// "stopped", not absent.
	Status string `json:"status"`
}

func (s *Server) handleListWorlds(w http.ResponseWriter, r *http.Request) {
	if pm, ok := loadProjectManifest(); ok {
		running := map[string]bool{}
		if liveWorlds, err := s.rstate.List(); err == nil {
			for _, lw := range liveWorlds {
				if lw.Config != "" {
					running[lw.Config] = true
				}
			}
		}
		out := make([]declaredWorldItem, 0, len(pm.Worlds))
		for name, def := range pm.Worlds {
			status := "stopped"
			if running[name] {
				status = "running"
			}
			out = append(out, declaredWorldItem{
				Name:       name,
				Agents:     def.Agents,
				Workspaces: def.Workspaces,
				Deps:       def.Deps,
				Status:     status,
			})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		jsonOK(w, out)
		return
	}

	worlds, err := s.rstate.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, worlds)
}

type agentListItem struct {
	Name   string              `json:"name"`
	Path   string              `json:"path"`
	Team   string              `json:"team,omitempty"`
	Role   string              `json:"role"`
	Layers map[string][]string `json:"layers"`
	// World is the name of the world in spwn.yaml that deploys this
	// agent, or empty when the agent isn't referenced by any world
	// (orphan) or no project is active.
	World string `json:"world,omitempty"`
	// Status is the agent's live state:
	//   "running":  its world is currently up
	//   "stopped":  referenced by a world in spwn.yaml that's down
	//   "orphan":   on-disk agent dir not referenced by any world
	Status string `json:"status"`
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := agentpkg.ListAgents()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	agentWorld := map[string]string{}
	runningWorlds := map[string]bool{}
	if pm, ok := loadProjectManifest(); ok {
		for wname, world := range pm.Worlds {
			for _, a := range world.Agents {
				agentWorld[a] = wname
			}
		}
	}
	if liveWorlds, lerr := s.rstate.List(); lerr == nil {
		for _, lw := range liveWorlds {
			if lw.Config != "" {
				runningWorlds[lw.Config] = true
			}
		}
	}

	result := make([]agentListItem, 0, len(agents))
	for _, a := range agents {
		role := "worker"
		manifestPath := filepath.Join(a.Path, "agent.yaml")
		if data, readErr := os.ReadFile(manifestPath); readErr == nil {
			var p agentYAML
			if yamlErr := yaml.Unmarshal(data, &p); yamlErr == nil && p.Role != "" {
				role = p.Role
			}
		}
		wname := agentWorld[a.Name]
		status := "orphan"
		if wname != "" {
			if runningWorlds[wname] {
				status = "running"
			} else {
				status = "stopped"
			}
		}
		result = append(result, agentListItem{
			Name:   a.Name,
			Path:   a.Path,
			Team:   a.Team,
			Role:   role,
			Layers: a.Layers,
			World:  wname,
			Status: status,
		})
	}

	jsonOK(w, result)
}

type projectManifest struct {
	Version int
	Name    string
	Worlds  map[string]projectWorldDef
}

type projectWorldDef struct {
	Agents     []string
	Workspaces []string
	Deps       []string
}

func loadInlineWorldManifest(name string) (models.Manifest, bool) {
	root := platform.ProjectRoot()
	if root == "" {
		return models.Manifest{}, false
	}
	p, err := projectpkg.Find(root)
	if err != nil || p == nil || p.Manifest == nil {
		return models.Manifest{}, false
	}
	w, exists := p.Manifest.Worlds[name]
	if !exists {
		return models.Manifest{}, false
	}

	seen := map[string]struct{}{}
	var deps []string
	add := func(t string) {
		if t == "" {
			return
		}
		if _, dup := seen[t]; dup {
			return
		}
		seen[t] = struct{}{}
		deps = append(deps, t)
	}
	for _, d := range p.Manifest.Deps {
		add(d)
	}
	agentPath := map[string]string{}
	for _, a := range p.Agents {
		agentPath[a.Name] = a.Path
	}
	for _, a := range p.OrphanAgents {
		agentPath[a.Name] = a.Path
	}
	for _, aname := range w.Agents {
		dir, ok := agentPath[aname]
		if !ok {
			continue
		}
		am, aerr := worldpkg.LoadAgentManifest(dir)
		if aerr != nil || am == nil {
			continue
		}
		for _, d := range am.Deps {
			add(d)
		}
	}
	return models.Manifest{Deps: deps}, true
}

func loadProjectManifest() (projectManifest, bool) {
	if platform.ProjectRoot() == "" {
		return projectManifest{}, false
	}
	p, err := projectpkg.Find(platform.ProjectRoot())
	if err != nil || p == nil || p.Manifest == nil {
		return projectManifest{}, false
	}
	out := projectManifest{
		Version: p.Manifest.Version,
		Name:    p.Manifest.Name,
		Worlds:  make(map[string]projectWorldDef, len(p.Manifest.Worlds)),
	}
	for name, w := range p.Manifest.Worlds {
		out.Worlds[name] = projectWorldDef{
			Agents:     append([]string(nil), w.Agents...),
			Workspaces: append([]string(nil), w.Workspaces...),
		}
	}
	return out, true
}

type agentYAML struct {
	Role    string `yaml:"role,omitempty" json:"role,omitempty"`
	Team    string `yaml:"team,omitempty" json:"team,omitempty"`
	Runtime struct {
		Engine   string `yaml:"engine,omitempty" json:"engine,omitempty"`
		Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
		Model    string `yaml:"model,omitempty" json:"model,omitempty"`
	} `yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

type agentProfileResponse struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	Role     string              `json:"role"`
	Team     string              `json:"team,omitempty"`
	Engine   string              `json:"engine"`
	Provider string              `json:"provider"`
	Purpose  string              `json:"purpose"`
	Profile  string              `json:"profile"`
	Traits   []string            `json:"traits"`
	Journal  []journalEntry      `json:"journal"`
	Layers   map[string][]string `json:"layers"`
}

type journalEntry struct {
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

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
		line = strings.TrimLeft(line, "-*• ")
		line = strings.TrimSpace(line)
		if line != "" {
			traits = append(traits, line)
		}
	}
	return traits
}

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

	role := "worker"
	engine := ""
	provider := ""
	manifestPath := filepath.Join(mindPath, "agent.yaml")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var p agentYAML
		if err := yaml.Unmarshal(data, &p); err == nil {
			if p.Role != "" {
				role = p.Role
			}
			if p.Runtime.Engine != "" {
				engine = p.Runtime.Engine
			}
			if p.Runtime.Provider != "" {
				provider = p.Runtime.Provider
			}
		}
	}

	purpose := readFirstLineContent(filepath.Join(mindPath, "identity", "purpose.md"))
	profileText := readFirstLineContent(filepath.Join(mindPath, "identity", "profile.md"))
	traits := parseTraits(filepath.Join(mindPath, "identity", "traits.md"))
	if traits == nil {
		traits = []string{}
	}

	journal := parseJournalFiles(filepath.Join(mindPath, "journal"))
	if journal == nil {
		journal = []journalEntry{}
	}

	resp := agentProfileResponse{
		Name:     info.Name,
		Path:     info.Path,
		Role:     role,
		Team:     info.Team,
		Engine:   engine,
		Provider: provider,
		Purpose:  purpose,
		Profile:  profileText,
		Traits:   traits,
		Journal:  journal,
		Layers:   info.Layers,
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

type agentSpec struct {
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type createWorldRequest struct {
	ConfigName string             `json:"config"`
	Name       string             `json:"name"`
	AgentName  string             `json:"agent"`
	Agents     []agentSpec        `json:"agents"`
	Workspaces []models.Workspace `json:"workspaces"`
	// Legacy single-workspace field; accepted for backward compatibility.
	Workspace string `json:"workspace"`
}

func (req createWorldRequest) resolveWorkspaces() []models.Workspace {
	if len(req.Workspaces) == 0 && req.Workspace != "" {
		return []models.Workspace{{Name: "default", Path: req.Workspace}}
	}
	return req.Workspaces
}

func (req createWorldRequest) resolveAgents() []architect.AgentSpec {
	if len(req.Agents) > 0 {
		out := make([]architect.AgentSpec, 0, len(req.Agents))
		for _, a := range req.Agents {
			if a.Name == "" {
				continue
			}
			out = append(out, architect.AgentSpec{Name: a.Name, Role: a.Role})
		}
		return out
	}
	return nil
}

func (s *Server) handleCreateWorld(w http.ResponseWriter, r *http.Request) {
	if !s.requireArch(w) {
		return
	}

	var body createWorldRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	workspaces := body.resolveWorkspaces()

	cfgName := body.ConfigName
	if cfgName == "" {
		cfgName = "default"
	}
	m, ok := loadInlineWorldManifest(cfgName)
	if !ok {
		var err error
		m, err = manifest.Load(cfgName)
		if err != nil {
			jsonError(w, "config not found: "+err.Error(), 400)
			return
		}
	}
	manifest.ApplyDefaults(&m)

	if os.Getenv("SPWN_SKIP_AUTH_VALIDATION") == "" {
		creds := authentication.ResolveAll()
		hasProvider := false
		for _, cred := range creds {
			if cred.Type != authentication.CredTypeNone {
				hasProvider = true
				break
			}
		}
		if !hasProvider {
			jsonError(w, "No AI provider configured. Go to Settings to connect an API key or subscription.", 400)
			return
		}
	}

	result, err := s.arch.Spawn(r.Context(), architect.SpawnOpts{
		ConfigName: cfgName,
		Name:       body.Name,
		AgentName:  body.AgentName,
		Agents:     body.resolveAgents(),
		Workspaces: workspaces,
		Manifest:   m,
	})
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, result)
}

func (s *Server) handleRenameWorld(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	if err := s.arch.Rename(r.Context(), worldID, body.Name); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	world, err := s.arch.Inspect(r.Context(), worldID)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, world)
}

func (s *Server) handleDeployAgent(w http.ResponseWriter, r *http.Request) {
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
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "agent name is required", 400)
		return
	}
	if err := s.arch.DeployAgent(r.Context(), worldID, body.Name, body.Role); err != nil {
		status := 500
		if err.Error() == fmt.Sprintf("agent %q is already deployed in world %s", body.Name, worldID) {
			status = 409
		}
		jsonError(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, map[string]string{"status": "ok", "agent": body.Name, "world": worldID})
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
	_ = json.NewDecoder(r.Body).Decode(&body)

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
		Agent   string `json:"agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		jsonError(w, "message is required", 400)
		return
	}

	worlds, err := s.rstate.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	agentName := body.Agent
	worldFound := false
	for _, u := range worlds {
		if u.ID == worldID {
			worldFound = true
			if agentName == "" {
				agentName = u.Agent
			}
			if agentName == "" && len(u.Agents) > 0 {
				agentName = u.Agents[0].Name
			}
			break
		}
	}
	if !worldFound {
		jsonError(w, "world not found", 404)
		return
	}
	if agentName == "" {
		jsonError(w, "world has no agent", 404)
		return
	}

	cmd := exec.CommandContext(r.Context(), "spwn", "agent", "talk", agentName, body.Message,
		"--world", worldID,
		"--output-format", "stream-json", "--verbose")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		jsonError(w, "failed to create stdout pipe: "+err.Error(), 500)
		return
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		jsonError(w, "failed to start agent talk: "+err.Error(), 500)
		return
	}

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

	_ = cmd.Wait()
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

	worlds, _ := s.rstate.List()
	agents, _ := agentpkg.ListAgents()

	tasksPending := 0
	tasksCompleted := 0
	stackPath := s.architectStackPath()
	if data, err := os.ReadFile(stackPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
				tasksCompleted++
			} else if strings.HasPrefix(trimmed, "- [ ]") {
				tasksPending++
			}
		}
	}

	resp := map[string]interface{}{
		"status":      status,
		"containerId": containerID,
		"uptime":      uptime,
		"kpis": map[string]interface{}{
			"worlds":         len(worlds),
			"agents":         len(agents),
			"tasksPending":   tasksPending,
			"tasksCompleted": tasksCompleted,
		},
	}

	s.architectMu.Lock()
	if s.architectSpawnState != nil {
		resp["progress"] = s.architectSpawnState.snapshot()
	}
	s.architectMu.Unlock()

	jsonOK(w, resp)
}

func (s *Server) handleArchitectLogs(w http.ResponseWriter, r *http.Request) {
	follow := r.URL.Query().Get("follow") == "true"
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, "--tail", tail, platform.ArchitectContainerName())

	cmd := exec.CommandContext(r.Context(), "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		jsonError(w, "stdout pipe: "+err.Error(), 500)
		return
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		jsonError(w, "start docker logs: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		if flusher != nil {
			flusher.Flush()
		}
	}
	_ = cmd.Wait()
	fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func (s *Server) architectStackPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	spwnHome := os.Getenv("SPWN_HOME")
	if spwnHome == "" {
		spwnHome = filepath.Join(home, ".spwn")
	}
	return filepath.Join(spwnHome, "architect", "stack.md")
}

func (s *Server) handleArchitectStackGet(w http.ResponseWriter, r *http.Request) {
	stackPath := s.architectStackPath()
	if stackPath == "" {
		jsonError(w, "could not determine stack path", 500)
		return
	}

	data, err := os.ReadFile(stackPath)
	if err != nil {
		defaultContent := "# Architect Stack\n\n## Focus\n\n## Queued\n\n## Done\n"
		jsonOK(w, map[string]string{"content": defaultContent})
		return
	}

	jsonOK(w, map[string]string{"content": string(data)})
}

func (s *Server) handleArchitectStackUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}

	stackPath := s.architectStackPath()
	if stackPath == "" {
		jsonError(w, "could not determine stack path", 500)
		return
	}

	if err := os.MkdirAll(filepath.Dir(stackPath), 0755); err != nil {
		jsonError(w, "failed to create directory: "+err.Error(), 500)
		return
	}

	if err := os.WriteFile(stackPath, []byte(body.Content), 0644); err != nil {
		jsonError(w, "failed to write stack file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleArchitectStart(w http.ResponseWriter, r *http.Request) {
	checkCmd := exec.CommandContext(r.Context(), "docker", "inspect", "--format", "{{.State.Running}}", "spwn-architect")
	if out, err := checkCmd.Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
		jsonOK(w, map[string]string{"status": "running", "message": "already running"})
		return
	}

	if err := s.startArchitectAsync(os.Getenv("SPWN_ARCHITECT_IMAGE")); err != nil {
		jsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	jsonOK(w, map[string]string{
		"status":  "starting",
		"message": "spawning architect - poll /api/architect/status for progress",
	})
}

func (s *Server) handleArchitectStop(w http.ResponseWriter, r *http.Request) {
	cmd := exec.CommandContext(r.Context(), "docker", "stop", "spwn-architect")
	if output, err := cmd.CombinedOutput(); err != nil {
		jsonError(w, fmt.Sprintf("failed to stop architect container: %s (%s)", strings.TrimSpace(string(output)), err), 500)
		return
	}
	jsonOK(w, map[string]string{"status": "stopped"})
}

// StackAction represents a parsed stack action from the architect's response.
type StackAction struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

func parseStackAction(text string) *StackAction {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		var actionType string
		var prefix string
		switch {
		case strings.HasPrefix(trimmed, "[STACK_PUSH]"):
			actionType = "push"
			prefix = "[STACK_PUSH]"
		case strings.HasPrefix(trimmed, "[STACK_POP]"):
			actionType = "pop"
			prefix = "[STACK_POP]"
		case strings.HasPrefix(trimmed, "[STACK_UPDATE]"):
			actionType = "update"
			prefix = "[STACK_UPDATE]"
		default:
			continue
		}

		title := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		action := &StackAction{
			Type:  actionType,
			Title: title,
		}

		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				break
			}
			if strings.HasPrefix(next, "Priority:") {
				action.Priority = strings.TrimSpace(strings.TrimPrefix(next, "Priority:"))
			} else if strings.HasPrefix(next, "Done:") {
				action.Description = strings.TrimSpace(strings.TrimPrefix(next, "Done:"))
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

	containerName := "spwn-architect"
	checkCmd := exec.CommandContext(r.Context(), "docker", "inspect", "--format", "{{.State.Status}}", containerName)
	checkOut, checkErr := checkCmd.Output()
	isRunning := checkErr == nil && strings.TrimSpace(string(checkOut)) == "running"

	if !isRunning {
		bgCtx := context.Background()
		startCmd := exec.CommandContext(bgCtx, "docker", "start", containerName)
		if startErr := startCmd.Run(); startErr != nil {
			spwnCmd := exec.CommandContext(bgCtx, "spwn", "architect", "start")
			if spwnErr := spwnCmd.Run(); spwnErr != nil {
				jsonError(w, "failed to start architect: "+spwnErr.Error(), 500)
				return
			}
		}
		time.Sleep(3 * time.Second)
	}

	dockerArgs := []string{
		"exec", "-u", "architect", "-w", "/me",
		"-e", "SPWN_HOME=/home/spwn/.spwn",
	}
	dockerArgs = append(dockerArgs, architect.DockerEnvArgs()...)
	dockerArgs = append(dockerArgs, containerName,
		"claude", "--dangerously-skip-permissions",
		"-p", body.Message,
		"--output-format", "stream-json", "--verbose",
		"--append-system-prompt",
		"You are the Architect. Read /me/ARCHITECT.md for your identity. "+
			"IMPORTANT: When a user asks you to do something, you MUST include a [STACK_PUSH] marker in your response. "+
			"Format: [STACK_PUSH] Short task title\nPriority: blocking|queued\nBrief description. "+
			"Also update /me/stack.md with the new task. "+
			"When completing a task use [STACK_POP] Short task title. "+
			"Read /me/skills/ for detailed guides.",
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

	_ = cmd.Wait()
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func (s *Server) getWorldContainerID(worldID string) (string, error) {
	worlds, err := s.rstate.List()
	if err != nil {
		return "", fmt.Errorf("failed to list worlds: %w", err)
	}
	for _, u := range worlds {
		if u.ID == worldID {
			if u.ContainerID == "" {
				return "", fmt.Errorf("world %s has no container", worldID)
			}
			return u.ContainerID, nil
		}
	}
	return "", fmt.Errorf("world not found: %s", worldID)
}

func dockerExecOutput(ctx context.Context, containerID string, args ...string) (string, error) {
	cmdArgs := append([]string{"exec", containerID}, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	out, err := cmd.Output()
	return string(out), err
}

func decodeJSONBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("empty request body")
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func sanitizeKnowledgePath(relPath string) (string, bool) {
	if relPath == "" {
		return "", false
	}
	if strings.HasPrefix(relPath, "/") {
		return "", false
	}
	cleaned := path.Clean(relPath)
	if cleaned == "." {
		return "", false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func (s *Server) handleWorldKnowledgeList(w http.ResponseWriter, r *http.Request) {
	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	containerID, err := s.getWorldContainerID(worldID)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	out, err := dockerExecOutput(r.Context(), containerID,
		"find", "/world/knowledge/", "-type", "f", "-not", "-name", ".*",
		"-printf", "%P\\t%s\\t%T@\\n")
	if err != nil {
		jsonOK(w, map[string]interface{}{"files": []interface{}{}})
		return
	}

	type fileEntry struct {
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		Modified string `json:"modified"`
	}

	var files []fileEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		size, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		epoch, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue
		}
		modified := time.Unix(int64(epoch), 0).Format(time.RFC3339)

		files = append(files, fileEntry{
			Path:     parts[0],
			Size:     size,
			Modified: modified,
		})
	}

	if files == nil {
		files = []fileEntry{}
	}

	jsonOK(w, map[string]interface{}{"files": files})
}

func (s *Server) handleWorldKnowledgeRead(w http.ResponseWriter, r *http.Request) {
	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	relPath := r.PathValue("path")
	if relPath == "" {
		jsonError(w, "file path is required", 400)
		return
	}

	safe, ok := sanitizeKnowledgePath(relPath)
	if !ok {
		jsonError(w, "invalid path", 400)
		return
	}

	containerID, err := s.getWorldContainerID(worldID)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	fullPath := "/world/knowledge/" + safe
	out, err := dockerExecOutput(r.Context(), containerID, "cat", fullPath)
	if err != nil {
		jsonError(w, "file not found: "+safe, 404)
		return
	}

	jsonOK(w, map[string]string{"path": safe, "content": out})
}

func (s *Server) handleWorldKnowledgeWrite(w http.ResponseWriter, r *http.Request) {
	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	relPath := r.PathValue("path")
	if relPath == "" {
		jsonError(w, "file path is required", 400)
		return
	}

	safe, ok := sanitizeKnowledgePath(relPath)
	if !ok {
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

	containerID, err := s.getWorldContainerID(worldID)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}

	fullPath := "/world/knowledge/" + safe

	dir := filepath.Dir(fullPath)
	if _, err := dockerExecOutput(r.Context(), containerID, "mkdir", "-p", dir); err != nil {
		jsonError(w, "failed to create directory: "+err.Error(), 500)
		return
	}

	cmdArgs := []string{"exec", "-i", containerID, "tee", fullPath}
	cmd := exec.CommandContext(r.Context(), "docker", cmdArgs...)
	cmd.Stdin = strings.NewReader(body.Content)
	if err := cmd.Run(); err != nil {
		jsonError(w, "failed to write file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "path": relPath})
}

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

	if body.Field == "team" {
		if err := projectpkg.SetAgentTeam(name, body.Content); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonOK(w, map[string]string{"status": "ok", "field": "team"})
		return
	}

	allowed := map[string]bool{"purpose": true, "profile": true, "traits": true}
	if !allowed[body.Field] {
		jsonError(w, "invalid field: must be one of purpose, profile, traits, team", 400)
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

	heading := strings.ToUpper(body.Field[:1]) + body.Field[1:]
	content := fmt.Sprintf("# %s\n\n%s\n", heading, body.Content)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		jsonError(w, "failed to write file: "+err.Error(), 500)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "field": body.Field})
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := projectpkg.ListTeams()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if teams == nil {
		teams = []projectpkg.Team{}
	}
	type teamWithMembers struct {
		projectpkg.Team
		Members []string `json:"members"`
	}
	result := make([]teamWithMembers, 0, len(teams))
	for _, t := range teams {
		members, _ := projectpkg.TeamMembers(t.Slug)
		if members == nil {
			members = []string{}
		}
		result = append(result, teamWithMembers{Team: t, Members: members})
	}
	jsonOK(w, result)
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var body projectpkg.Team
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}
	if body.Name == "" {
		jsonError(w, "team name is required", 400)
		return
	}
	if body.Slug == "" {
		body.Slug = projectpkg.Slugify(body.Name)
	}
	if err := projectpkg.CreateTeam(body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, body)
}

func (s *Server) handleUpdateTeam(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "team slug is required", 400)
		return
	}
	existing, err := projectpkg.GetTeam(slug)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	var body projectpkg.Team
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}
	if body.Name != "" {
		existing.Name = body.Name
	}
	if body.Color != "" {
		existing.Color = body.Color
	}
	if body.Description != "" {
		existing.Description = body.Description
	}
	if err := projectpkg.UpdateTeam(*existing); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, existing)
}

func (s *Server) handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "team slug is required", 400)
		return
	}
	if err := projectpkg.DeleteTeam(slug); err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	organizations, err := projectpkg.ListOrganizations()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if organizations == nil {
		organizations = []projectpkg.Organization{}
	}
	jsonOK(w, organizations)
}

func (s *Server) handleGetOrganization(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "organization slug is required", 400)
		return
	}
	h, err := projectpkg.GetOrganization(slug)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	jsonOK(w, h)
}

func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	var body projectpkg.Organization
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}
	if body.Name == "" {
		jsonError(w, "organization name is required", 400)
		return
	}
	if body.Slug == "" {
		body.Slug = projectpkg.Slugify(body.Name)
	}
	if err := projectpkg.CreateOrganization(body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, body)
}

func (s *Server) handleUpdateOrganization(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "organization slug is required", 400)
		return
	}
	existing, err := projectpkg.GetOrganization(slug)
	if err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	var body projectpkg.Organization
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), 400)
		return
	}
	if body.Name != "" {
		existing.Name = body.Name
	}
	if body.Description != "" {
		existing.Description = body.Description
	}
	if body.Roles != nil {
		existing.Roles = body.Roles
	}
	existing.Slug = slug
	if err := projectpkg.UpdateOrganization(*existing); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonOK(w, existing)
}

func (s *Server) handleDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "organization slug is required", 400)
		return
	}
	if slug == "default" {
		jsonError(w, "cannot delete the default organization", 400)
		return
	}
	if err := projectpkg.DeleteOrganization(slug); err != nil {
		jsonError(w, err.Error(), 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetAgentFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	filePath := r.PathValue("path")
	if name == "" || filePath == "" {
		jsonError(w, "agent name and file path are required", 400)
		return
	}

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

func (s *Server) handleAuthProviders(w http.ResponseWriter, r *http.Request) {
	creds := authentication.ResolveAll()

	type detectionInfo struct {
		Method         string `json:"method"`
		CredentialType string `json:"credentialType"`
		Source         string `json:"source"`
	}
	type providerInfo struct {
		Provider       string                    `json:"provider"`
		Connected      bool                      `json:"connected"`
		Disabled       bool                      `json:"disabled,omitempty"`
		Method         string                    `json:"method,omitempty"`
		ActiveMethod   string                    `json:"activeMethod,omitempty"`
		CredentialType string                    `json:"credentialType"`
		Source         string                    `json:"source"`
		Detections     []detectionInfo           `json:"detections,omitempty"`
		Error          string                    `json:"error,omitempty"`
		Plan           string                    `json:"plan,omitempty"`
		Usage          *authentication.UsageInfo `json:"usage,omitempty"`
	}

	var providers []providerInfo
	for p, cred := range creds {
		disabled := authentication.IsProviderDisabled(p)
		connected := cred.Type != authentication.CredTypeNone && !disabled
		info := providerInfo{
			Provider:       string(cred.Provider),
			CredentialType: string(cred.Type),
			Source:         cred.Source,
			Connected:      connected,
			Disabled:       disabled,
			Method:         string(cred.Method()),
			ActiveMethod:   string(authentication.ActiveMethod(p)),
		}
		for _, d := range authentication.DetectMethods(p) {
			info.Detections = append(info.Detections, detectionInfo{
				Method:         string(d.Method()),
				CredentialType: string(d.Type),
				Source:         d.Source,
			})
		}
		providers = append(providers, info)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Provider < providers[j].Provider })
	jsonOK(w, map[string]interface{}{"providers": providers})
}

func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
	}
	if err := decodeJSONBody(r, &body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	cred := authentication.Resolve(authentication.Provider(body.Provider))
	status := authentication.Validate(r.Context(), cred)
	jsonOK(w, status)
}

func (s *Server) handleAuthConfigure(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		Token    string `json:"token"`
	}
	if err := decodeJSONBody(r, &body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if body.Token == "" {
		jsonError(w, "token required", 400)
		return
	}
	if err := authentication.SaveToken(body.Token); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	if body.Provider != "" {
		_ = authentication.EnableProvider(authentication.Provider(body.Provider))
	}
	_ = authentication.SyncCredentials()
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthReconnect(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
	}
	if err := decodeJSONBody(r, &body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	if body.Provider != "" {
		_ = authentication.EnableProvider(authentication.Provider(body.Provider))
	}
	_ = authentication.SyncCredentials()
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthReset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
	}
	if err := decodeJSONBody(r, &body); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	result := &authentication.LogoutResult{}
	if body.Provider != "" {
		p := authentication.Provider(body.Provider)
		result = authentication.LogoutProvider(p, authentication.LogoutOpts{})
		_ = authentication.DisableProvider(p)
	}

	jsonOK(w, map[string]interface{}{
		"status":    "ok",
		"cleared":   result.Cleared,
		"remaining": result.Remaining,
	})
}

type historyMessage struct {
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Timestamp string  `json:"timestamp"`
	SessionID string  `json:"sessionId"`
	Type      string  `json:"type"`
	ToolName  string  `json:"toolName,omitempty"`
	Cost      float64 `json:"cost,omitempty"`
	Duration  int     `json:"durationMs,omitempty"`
}

type historySession struct {
	ID        string           `json:"id"`
	Messages  []historyMessage `json:"messages"`
	StartedAt string           `json:"startedAt"`
	Cost      float64          `json:"cost,omitempty"`
}

func (s *Server) parseJSONLSessions(ctx context.Context, container, user, basePath string, limit int) []historySession {
	lsCmd := exec.CommandContext(ctx, "docker", "exec", "-u", user,
		container, "ls", "-t", basePath)
	lsOut, err := lsCmd.Output()
	if err != nil {
		return nil
	}

	files := strings.Fields(strings.TrimSpace(string(lsOut)))
	if len(files) == 0 {
		return nil
	}
	if len(files) > limit {
		files = files[:limit]
	}

	var sessions []historySession
	for _, file := range files {
		if !strings.HasSuffix(file, ".jsonl") {
			continue
		}
		path := basePath + file
		catCmd := exec.CommandContext(ctx, "docker", "exec", "-u", user,
			container, "cat", path)
		data, err := catCmd.Output()
		if err != nil {
			continue
		}

		session := historySession{ID: strings.TrimSuffix(file, ".jsonl")}
		var msgs []historyMessage

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			eventType, _ := event["type"].(string)
			timestamp, _ := event["timestamp"].(string)
			sessionID, _ := event["sessionId"].(string)

			if session.StartedAt == "" && timestamp != "" {
				session.StartedAt = timestamp
			}

			switch eventType {
			case "user":
				msg, _ := event["message"].(map[string]interface{})
				content, _ := msg["content"].(string)
				if content != "" {
					msgs = append(msgs, historyMessage{
						Role: "user", Content: content,
						Timestamp: timestamp, SessionID: sessionID, Type: "text",
					})
				}
			case "assistant":
				msg, _ := event["message"].(map[string]interface{})
				contentArr, _ := msg["content"].([]interface{})
				for _, c := range contentArr {
					block, _ := c.(map[string]interface{})
					blockType, _ := block["type"].(string)
					switch blockType {
					case "text":
						text, _ := block["text"].(string)
						if text != "" {
							msgs = append(msgs, historyMessage{
								Role: "assistant", Content: text,
								Timestamp: timestamp, SessionID: sessionID, Type: "text",
							})
						}
					case "tool_use":
						toolName, _ := block["name"].(string)
						msgs = append(msgs, historyMessage{
							Role: "assistant", Type: "tool_use",
							ToolName: toolName, Timestamp: timestamp, SessionID: sessionID,
						})
					}
				}
			case "result":
				cost, _ := event["total_cost_usd"].(float64)
				dur, _ := event["duration_ms"].(float64)
				session.Cost = cost
				msgs = append(msgs, historyMessage{
					Role: "assistant", Type: "result",
					Cost: cost, Duration: int(dur),
					Timestamp: timestamp, SessionID: sessionID,
				})
			}
		}

		session.Messages = msgs
		if len(msgs) > 0 {
			sessions = append(sessions, session)
		}
	}

	for i, j := 0, len(sessions)-1; i < j; i, j = i+1, j-1 {
		sessions[i], sessions[j] = sessions[j], sessions[i]
	}

	return sessions
}

func (s *Server) handleArchitectHistory(w http.ResponseWriter, r *http.Request) {
	sessions := s.parseJSONLSessions(r.Context(),
		"spwn-architect", "architect",
		"/home/architect/.claude/projects/-world/", 5)
	if sessions == nil {
		sessions = []historySession{}
	}
	jsonOK(w, map[string]interface{}{"sessions": sessions})
}

func (s *Server) handleWorldHistory(w http.ResponseWriter, r *http.Request) {
	worldID := r.PathValue("id")
	if worldID == "" {
		jsonError(w, "world id is required", 400)
		return
	}

	worlds, err := s.rstate.List()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	var containerID string
	for _, u := range worlds {
		if u.ID == worldID {
			containerID = u.ContainerID
			break
		}
	}
	if containerID == "" {
		jsonError(w, "world not found", 404)
		return
	}

	sessions := s.parseJSONLSessions(r.Context(),
		containerID, "spwn",
		"/home/spwn/.claude/projects/-workspace/", 5)
	if sessions == nil {
		sessions = []historySession{}
	}
	jsonOK(w, map[string]interface{}{"sessions": sessions})
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := activity.ReadOpts{
		Limit:   50,
		Type:    activity.Type(q.Get("type")),
		WorldID: q.Get("world"),
		AgentID: q.Get("agent"),
		Actor:   q.Get("actor"),
	}
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			opts.Limit = n
		}
	}
	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = t
		}
	}

	events, err := activity.Read(opts)
	if err != nil {
		jsonError(w, "failed to read activity log: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []activity.Event{}
	}
	jsonOK(w, map[string]interface{}{"events": events})
}
