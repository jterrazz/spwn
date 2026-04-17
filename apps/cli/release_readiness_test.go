package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// spwnBinary is the absolute path to the real CLI binary used by every
// subtest below. Populated by TestMain via `go build`.
var spwnBinary string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "spwn-release-qa-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: mkdirtemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "spwn")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/spwn")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: build failed: %v\n", err)
		os.Exit(1)
	}
	spwnBinary = bin

	os.Exit(m.Run())
}

// runCLI runs the built binary with the given extra env, working dir,
// and args. Returns stdout, stderr, and the exit code (0 on success,
// >0 on failure). env is appended to the parent env; entries in env
// override earlier duplicates.
func runCLI(t *testing.T, env []string, wd string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(spwnBinary, args...)
	cmd.Dir = wd
	// Inherit PATH etc. but wipe anything that could leak host state
	// into the test — particularly a user's real SPWN_HOME.
	base := os.Environ()
	filtered := base[:0]
	for _, kv := range base {
		if strings.HasPrefix(kv, "SPWN_HOME=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	cmd.Env = append(filtered, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("runCLI: %v", err)
		}
	}
	return stdout.String(), stderr.String(), code
}

// freshEnv returns an env slice pointing SPWN_HOME at a fresh temp dir.
// Used by subtests to isolate user-level state.
func freshEnv(t *testing.T) (env []string, home string) {
	t.Helper()
	home = t.TempDir()
	return []string{"SPWN_HOME=" + home}, home
}

// mustInit scaffolds a fresh project at wd with the given name.
func mustInit(t *testing.T, env []string, wd, name string) {
	t.Helper()
	args := []string{"init"}
	if name != "" {
		args = append(args, "--name", name)
	}
	stdout, stderr, code := runCLI(t, env, wd, args...)
	if code != 0 {
		t.Fatalf("init failed: code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
}

// readFile is a thin wrapper around os.ReadFile that fails the test on
// error so call sites stay uncluttered.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestReleaseReadiness runs a 50-scenario pre-release QA audit against
// the real spwn binary. Each subtest is self-contained: fresh
// SPWN_HOME, fresh project dir, fresh env. The suite covers the six
// refactors shipped in the last week (knowledge-at-world, ref-syntax
// cleanup, VOLUME removal, config.yaml seeding, Destroy timeout, and
// image-generator cleanup).
func TestReleaseReadiness(t *testing.T) {
	// --------------------------------------------------------------------
	// A — init + scaffold
	// --------------------------------------------------------------------

	t.Run("01_init_name_flag_writes_manifest", func(t *testing.T) {
		// A01: `init --name acme` creates spwn.yaml containing that name.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		m := readFile(t, filepath.Join(wd, "spwn.yaml"))
		if !strings.Contains(m, "name: acme") {
			t.Fatalf("spwn.yaml missing `name: acme`:\n%s", m)
		}
	})

	t.Run("02_init_auto_names_from_dir", func(t *testing.T) {
		// A02: omitting --name uses the directory basename.
		t.Parallel()
		env, _ := freshEnv(t)
		parent := t.TempDir()
		wd := filepath.Join(parent, "myproj")
		if err := os.MkdirAll(wd, 0o755); err != nil {
			t.Fatal(err)
		}
		mustInit(t, env, wd, "")
		m := readFile(t, filepath.Join(wd, "spwn.yaml"))
		if !strings.Contains(m, "name: myproj") {
			t.Fatalf("spwn.yaml missing `name: myproj`:\n%s", m)
		}
	})

	t.Run("03_init_writes_4_layer_mind", func(t *testing.T) {
		// A03: fresh init writes identity/skills/playbooks/journal but NO knowledge.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		for _, layer := range []string{"identity", "skills", "playbooks", "journal"} {
			if _, err := os.Stat(filepath.Join(wd, "spwn/agents/neo", layer)); err != nil {
				t.Errorf("missing agent layer %s: %v", layer, err)
			}
		}
		if _, err := os.Stat(filepath.Join(wd, "spwn/agents/neo/knowledge")); !os.IsNotExist(err) {
			t.Errorf("agent knowledge layer should NOT exist, stat err=%v", err)
		}
	})

	t.Run("04_init_writes_world_knowledge", func(t *testing.T) {
		// A04: fresh init seeds spwn/worlds/neo/knowledge/.gitkeep.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		gk := filepath.Join(wd, "spwn/worlds/neo/knowledge/.gitkeep")
		if _, err := os.Stat(gk); err != nil {
			t.Fatalf("world knowledge .gitkeep missing: %v", err)
		}
	})

	t.Run("05_init_seeds_lockfile", func(t *testing.T) {
		// A05: spwn.lock is seeded with three spwn:* entries.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		lock := readFile(t, filepath.Join(wd, "spwn.lock"))
		for _, ref := range []string{"spwn:unix", "spwn:git", "spwn:python"} {
			if !strings.Contains(lock, ref) {
				t.Errorf("lockfile missing %q:\n%s", ref, lock)
			}
		}
	})

	t.Run("06_init_refuses_overwrite", func(t *testing.T) {
		// A06: re-running init without --force exits 1 when spwn.yaml exists.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, stderr, code := runCLI(t, env, wd, "init", "--name", "acme")
		if code == 0 {
			t.Fatalf("second init should fail, got exit 0.\nstderr=%s", stderr)
		}
	})

	t.Run("07_init_force_overwrites", func(t *testing.T) {
		// A07: `--force` replaces spwn.yaml with the new name.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, stderr, code := runCLI(t, env, wd, "init", "--name", "zeta", "--force")
		if code != 0 {
			t.Fatalf("init --force failed: code=%d\nstderr=%s", code, stderr)
		}
		m := readFile(t, filepath.Join(wd, "spwn.yaml"))
		if !strings.Contains(m, "name: zeta") {
			t.Fatalf("overwrite did not take effect:\n%s", m)
		}
	})

	t.Run("08_init_passes_check", func(t *testing.T) {
		// A08: fresh init is validation-clean (`spwn check` exits 0).
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code != 0 {
			t.Fatalf("check failed: code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
	})

	// --------------------------------------------------------------------
	// B — ref syntax + lockfile
	// --------------------------------------------------------------------

	t.Run("09_install_scheme_form_to_lock", func(t *testing.T) {
		// B09: scheme-form install lands in the lockfile.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "install", "spwn:python")
		if code != 0 {
			t.Fatalf("install failed: code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
		lock := readFile(t, filepath.Join(wd, "spwn.lock"))
		if !strings.Contains(lock, "spwn:python") {
			t.Fatalf("lockfile missing spwn:python:\n%s", lock)
		}
	})

	t.Run("10_agent_add_rejects_at_prefix", func(t *testing.T) {
		// B10: `@spwn/node` is malformed under the new ref grammar.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "agent", "add", "neo", "--dep", "@spwn/node")
		if code == 0 {
			t.Fatalf("agent add @spwn/node should fail, got exit 0.\nstdout=%s", stdout)
		}
		combined := stdout + stderr
		if !(strings.Contains(combined, "does not exist") || strings.Contains(combined, "unsupported") || strings.Contains(combined, "malformed")) {
			t.Fatalf("expected malformed/unsupported error, got:\n%s", combined)
		}
	})

	t.Run("11_check_reports_registry_unsupported", func(t *testing.T) {
		// B11: github: refs surface as "remote registries are not yet supported".
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "agent", "add", "neo", "--dep", "github:acme/x"); code != 0 {
			t.Fatalf("agent add github:acme/x should accept the ref, got exit %d", code)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should fail with github registry ref")
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "remote registries are not yet supported") {
			t.Fatalf("expected registry-unsupported message, got:\n%s", combined)
		}
	})

	t.Run("12_check_reports_unknown_local", func(t *testing.T) {
		// B12: bare local refs with no matching file surface as "does not exist".
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		// Append a bare ref that resolves to nothing on disk.
		yamlPath := filepath.Join(wd, "spwn/agents/neo/agent.yaml")
		y := readFile(t, yamlPath)
		if err := os.WriteFile(yamlPath, []byte(y+"\n  - unknown-local\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should fail for unknown bare ref")
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "unknown-local") || !strings.Contains(combined, "does not exist") {
			t.Fatalf("expected unknown-local not-found message, got:\n%s", combined)
		}
	})

	t.Run("13_lockfile_has_no_legacy_at", func(t *testing.T) {
		// B13: scheme-form install never writes `@spwn/` into the lockfile.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "install", "spwn:python"); code != 0 {
			t.Fatalf("install failed")
		}
		lock := readFile(t, filepath.Join(wd, "spwn.lock"))
		if strings.Contains(lock, "@spwn/") {
			t.Fatalf("lockfile contains legacy @spwn/ ref:\n%s", lock)
		}
	})

	t.Run("14_check_rejects_legacy_at_in_agent_yaml", func(t *testing.T) {
		// B14: `@spwn/name` written by hand into agent.yaml fails check.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		yamlPath := filepath.Join(wd, "spwn/agents/neo/agent.yaml")
		y := readFile(t, yamlPath)
		if err := os.WriteFile(yamlPath, []byte(y+"\n  - \"@spwn/python\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should reject legacy @spwn/python ref")
		}
	})

	// --------------------------------------------------------------------
	// C — agent CRUD
	// --------------------------------------------------------------------

	t.Run("15_agent_create_global_4_layer", func(t *testing.T) {
		// C15: user-mode `agent create` produces a 4-layer Mind (no knowledge).
		t.Parallel()
		env, home := freshEnv(t)
		// No project: operate in global mode from a fresh dir.
		wd := t.TempDir()
		if _, _, code := runCLI(t, env, wd, "agent", "create", "neo"); code != 0 {
			t.Fatalf("global agent create should succeed")
		}
		base := filepath.Join(home, "agents/neo")
		for _, layer := range []string{"identity", "skills", "playbooks", "journal"} {
			if _, err := os.Stat(filepath.Join(base, layer)); err != nil {
				t.Errorf("missing %s layer: %v", layer, err)
			}
		}
		if _, err := os.Stat(filepath.Join(base, "knowledge")); !os.IsNotExist(err) {
			t.Errorf("agent knowledge dir should NOT exist under user mode, stat err=%v", err)
		}
	})

	t.Run("16_agent_create_rejects_invalid_name", func(t *testing.T) {
		// C16: names with spaces are rejected up front.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, _, code := runCLI(t, env, wd, "agent", "create", "with space")
		if code == 0 {
			t.Fatalf("creating agent with space should fail")
		}
	})

	t.Run("17_agent_create_rejects_reserved_name", func(t *testing.T) {
		// C17: `ls` collides with `spwn agent ls` and must be rejected.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, stderr, code := runCLI(t, env, wd, "agent", "create", "ls")
		if code == 0 {
			t.Fatalf("creating agent ls should fail")
		}
		if !strings.Contains(stderr, "reserved") {
			t.Fatalf("expected reserved-name error, got:\n%s", stderr)
		}
	})

	t.Run("18_duplicate_agent_create_fails", func(t *testing.T) {
		// C18: creating the same agent twice errors without --force.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme") // init already makes `neo`
		_, _, code := runCLI(t, env, wd, "agent", "create", "neo")
		if code == 0 {
			t.Fatalf("duplicate create should fail")
		}
	})

	t.Run("19_agent_create_force_succeeds", func(t *testing.T) {
		// C19: `--force` re-scaffolds over an existing agent.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, stderr, code := runCLI(t, env, wd, "agent", "create", "neo", "--force")
		if code != 0 {
			t.Fatalf("force create should succeed: stderr=%s", stderr)
		}
	})

	t.Run("20_agent_ls_json_lists_agents", func(t *testing.T) {
		// C20: project-aware `agent ls --json` prints the roster.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "agent", "ls", "--json")
		if code != 0 {
			t.Fatalf("agent ls --json failed: stderr=%s", stderr)
		}
		if !strings.Contains(stdout, "\"name\": \"neo\"") {
			t.Fatalf("agent ls --json missing neo:\n%s", stdout)
		}
	})

	t.Run("21_agent_rm_deletes_dir", func(t *testing.T) {
		// C21: `agent rm` removes the agent directory from disk.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "agent", "rm", "neo"); code != 0 {
			t.Fatalf("agent rm failed")
		}
		if _, err := os.Stat(filepath.Join(wd, "spwn/agents/neo")); !os.IsNotExist(err) {
			t.Fatalf("agent dir still present after rm: err=%v", err)
		}
	})

	// --------------------------------------------------------------------
	// D — agent compose
	// --------------------------------------------------------------------

	t.Run("22_agent_add_appends_dep", func(t *testing.T) {
		// D22: `agent add --dep` appends to agent.yaml dependencies.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		// Remove the default python to prove add appends it.
		if _, _, code := runCLI(t, env, wd, "agent", "remove", "neo", "--dep", "spwn:python"); code != 0 {
			t.Fatalf("pre-remove python failed")
		}
		if _, _, code := runCLI(t, env, wd, "agent", "add", "neo", "--dep", "spwn:python"); code != 0 {
			t.Fatalf("agent add spwn:python failed")
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if !strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml missing spwn:python:\n%s", y)
		}
	})

	t.Run("23_agent_add_is_idempotent", func(t *testing.T) {
		// D23: adding the same dep twice still results in one entry.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		for i := 0; i < 2; i++ {
			if _, _, code := runCLI(t, env, wd, "agent", "add", "neo", "--dep", "spwn:node"); code != 0 {
				t.Fatalf("iter %d add failed", i)
			}
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if strings.Count(y, "spwn:node") != 1 {
			t.Fatalf("expected exactly 1 occurrence of spwn:node:\n%s", y)
		}
	})

	t.Run("24_agent_remove_detaches", func(t *testing.T) {
		// D24: `agent remove --dep` drops the entry.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "agent", "remove", "neo", "--dep", "spwn:python"); code != 0 {
			t.Fatalf("agent remove failed")
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml still has spwn:python:\n%s", y)
		}
	})

	t.Run("25_agent_remove_absent_errors", func(t *testing.T) {
		// D25: removing an un-attached dep must error, not silently no-op.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, _, code := runCLI(t, env, wd, "agent", "remove", "neo", "--dep", "spwn:never-added")
		if code == 0 {
			t.Fatalf("remove of absent dep should error")
		}
	})

	// --------------------------------------------------------------------
	// E — install / uninstall
	// --------------------------------------------------------------------

	t.Run("26_install_updates_agents_and_lockfile", func(t *testing.T) {
		// E26: install adds the dep to every agent and pins the lockfile.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		// First remove the default entry so we can prove install re-adds it.
		if _, _, code := runCLI(t, env, wd, "agent", "remove", "neo", "--dep", "spwn:python"); code != 0 {
			t.Fatalf("pre-remove failed")
		}
		if _, _, code := runCLI(t, env, wd, "install", "spwn:python"); code != 0 {
			t.Fatalf("install failed")
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if !strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml missing spwn:python:\n%s", y)
		}
		lock := readFile(t, filepath.Join(wd, "spwn.lock"))
		if !strings.Contains(lock, "spwn:python") {
			t.Fatalf("lockfile missing spwn:python:\n%s", lock)
		}
	})

	t.Run("27_install_idempotent", func(t *testing.T) {
		// E27: install twice keeps a single agent.yaml entry.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		for i := 0; i < 2; i++ {
			if _, _, code := runCLI(t, env, wd, "install", "spwn:node"); code != 0 {
				t.Fatalf("install iter %d failed", i)
			}
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if strings.Count(y, "spwn:node") != 1 {
			t.Fatalf("expected 1 spwn:node, got:\n%s", y)
		}
	})

	t.Run("28_uninstall_removes_from_both_sides", func(t *testing.T) {
		// E28: uninstall strips both agent.yaml and the lockfile.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "uninstall", "spwn:python"); code != 0 {
			t.Fatalf("uninstall failed")
		}
		y := readFile(t, filepath.Join(wd, "spwn/agents/neo/agent.yaml"))
		if strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml still references spwn:python:\n%s", y)
		}
		lock := readFile(t, filepath.Join(wd, "spwn.lock"))
		if strings.Contains(lock, "spwn:python") {
			t.Fatalf("lockfile still references spwn:python:\n%s", lock)
		}
	})

	t.Run("29_install_rejects_at_prefix", func(t *testing.T) {
		// E29: `@spwn/python` is no longer valid input to install.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		_, _, code := runCLI(t, env, wd, "install", "@spwn/python")
		if code == 0 {
			t.Fatalf("install @spwn/python should be rejected")
		}
	})

	// --------------------------------------------------------------------
	// F — skill / tool / hook verbs
	// --------------------------------------------------------------------

	t.Run("30_skill_new_writes_frontmatter", func(t *testing.T) {
		// F30: `skill new` authors a file with YAML frontmatter.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, stderr, code := runCLI(t, env, wd, "skill", "new", "reading"); code != 0 {
			t.Fatalf("skill new failed: stderr=%s", stderr)
		}
		path := filepath.Join(wd, "spwn/skills/reading.md")
		body := readFile(t, path)
		if !strings.HasPrefix(body, "---\n") {
			t.Fatalf("skill file missing leading frontmatter:\n%s", body)
		}
		if !strings.Contains(body, "name: reading") || !strings.Contains(body, "description:") {
			t.Fatalf("frontmatter missing name/description:\n%s", body)
		}
	})

	t.Run("31_skill_ls_includes_new_skill", func(t *testing.T) {
		// F31: after authoring, `skill ls` lists the slug.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "skill", "new", "reading"); code != 0 {
			t.Fatalf("skill new failed")
		}
		stdout, stderr, code := runCLI(t, env, wd, "skill", "ls")
		if code != 0 {
			t.Fatalf("skill ls failed: stderr=%s", stderr)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "reading") {
			t.Fatalf("skill ls output missing `reading`:\n%s", combined)
		}
	})

	t.Run("32_skill_rm_removes_file", func(t *testing.T) {
		// F32: `skill rm` deletes the underlying markdown file.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "skill", "new", "reading"); code != 0 {
			t.Fatalf("skill new failed")
		}
		if _, _, code := runCLI(t, env, wd, "skill", "rm", "reading"); code != 0 {
			t.Fatalf("skill rm failed")
		}
		if _, err := os.Stat(filepath.Join(wd, "spwn/skills/reading.md")); !os.IsNotExist(err) {
			t.Fatalf("skill file should be gone, err=%v", err)
		}
	})

	t.Run("33_bare_local_ref_passes_check", func(t *testing.T) {
		// F33: a bare local ref resolves to spwn/skills/<name>.md and `check` stays clean.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "skill", "new", "reading"); code != 0 {
			t.Fatalf("skill new failed")
		}
		yamlPath := filepath.Join(wd, "spwn/agents/neo/agent.yaml")
		y := readFile(t, yamlPath)
		if err := os.WriteFile(yamlPath, []byte(y+"\n  - reading\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code != 0 {
			t.Fatalf("check should pass with bare local ref, got exit %d:\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
	})

	// --------------------------------------------------------------------
	// G — help text consistency
	// --------------------------------------------------------------------

	t.Run("34_agent_help_mentions_4_layer", func(t *testing.T) {
		// G34: `agent --help` describes the new 4-layer Mind.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, stderr, code := runCLI(t, env, wd, "agent", "--help")
		if code != 0 {
			t.Fatalf("agent --help failed: stderr=%s", stderr)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "4-layer") {
			t.Fatalf("agent --help missing `4-layer`:\n%s", combined)
		}
		if !strings.Contains(combined, "identity/skills/playbooks/journal") {
			t.Fatalf("agent --help missing layer list:\n%s", combined)
		}
	})

	t.Run("35_agent_create_help_mentions_4_layer", func(t *testing.T) {
		// G35: `agent create --help` mentions the 4-layer Mind.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, stderr, code := runCLI(t, env, wd, "agent", "create", "--help")
		if code != 0 {
			t.Fatalf("agent create --help failed: stderr=%s", stderr)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "4-layer Mind") {
			t.Fatalf("agent create --help missing `4-layer Mind`:\n%s", combined)
		}
	})

	t.Run("36_no_legacy_at_in_help_outputs", func(t *testing.T) {
		// G36: the old `@spwn/` substring is gone from the top-level help.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		for _, args := range [][]string{{"--help"}, {"agent", "--help"}, {"install", "--help"}} {
			out, errOut, code := runCLI(t, env, wd, args...)
			if code != 0 {
				t.Fatalf("%v exit %d: stderr=%s", args, code, errOut)
			}
			combined := out + errOut
			if strings.Contains(combined, "@spwn/") {
				t.Fatalf("%v help still contains `@spwn/`:\n%s", args, combined)
			}
		}
	})

	t.Run("37_no_stale_layer_counts_in_help", func(t *testing.T) {
		// G37: no references to the old 5/6-layer Mind remain in help text.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		for _, args := range [][]string{{"--help"}, {"agent", "--help"}, {"install", "--help"}} {
			out, errOut, code := runCLI(t, env, wd, args...)
			if code != 0 {
				t.Fatalf("%v exit %d", args, code)
			}
			combined := out + errOut
			for _, bad := range []string{"5-layer", "6-layer"} {
				if strings.Contains(combined, bad) {
					t.Fatalf("%v help contains stale %q:\n%s", args, bad, combined)
				}
			}
		}
	})

	t.Run("38_check_help_runs", func(t *testing.T) {
		// G38: `check --help` renders and exits cleanly.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, stderr, code := runCLI(t, env, wd, "check", "--help")
		if code != 0 {
			t.Fatalf("check --help exit %d: stderr=%s", code, stderr)
		}
		if !strings.Contains(stdout, "check") {
			t.Fatalf("check --help output suspiciously empty:\n%s", stdout)
		}
	})

	// --------------------------------------------------------------------
	// H — config.yaml + migrations
	// --------------------------------------------------------------------

	t.Run("39_first_invocation_creates_config_yaml", func(t *testing.T) {
		// H39: a fresh SPWN_HOME grows config.yaml on the first CLI call.
		t.Parallel()
		env, home := freshEnv(t)
		wd := t.TempDir()
		_, _, code := runCLI(t, env, wd, "auth", "status")
		if code != 0 {
			t.Fatalf("auth status failed with exit %d", code)
		}
		cfg := filepath.Join(home, "config.yaml")
		if _, err := os.Stat(cfg); err != nil {
			t.Fatalf("~/.spwn/config.yaml should exist after first invocation: %v", err)
		}
	})

	t.Run("40_config_yaml_has_expected_fields", func(t *testing.T) {
		// H40: the seeded config carries apiVersion + runtime + telemetry + update.
		t.Parallel()
		env, home := freshEnv(t)
		wd := t.TempDir()
		if _, _, code := runCLI(t, env, wd, "auth", "status"); code != 0 {
			t.Fatalf("auth status failed")
		}
		cfg := readFile(t, filepath.Join(home, "config.yaml"))
		for _, need := range []string{"apiVersion: spwn/v2", "default_backend: spwn:claude-code", "telemetry", "update"} {
			if !strings.Contains(cfg, need) {
				t.Fatalf("config.yaml missing %q:\n%s", need, cfg)
			}
		}
	})

	t.Run("41_existing_config_is_preserved", func(t *testing.T) {
		// H41: we never clobber a user-edited config.yaml.
		t.Parallel()
		env, home := freshEnv(t)
		wd := t.TempDir()
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatal(err)
		}
		custom := "apiVersion: spwn/v2\n# hand-edited\ntelemetry:\n  enabled: true\n"
		cfgPath := filepath.Join(home, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte(custom), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, code := runCLI(t, env, wd, "auth", "status"); code != 0 {
			t.Fatalf("auth status failed")
		}
		got := readFile(t, cfgPath)
		if got != custom {
			t.Fatalf("config.yaml was modified:\nbefore=%q\nafter =%q", custom, got)
		}
	})

	t.Run("42_malformed_config_does_not_crash", func(t *testing.T) {
		// H42: a malformed config must not abort --version with exit > 1.
		t.Parallel()
		env, home := freshEnv(t)
		wd := t.TempDir()
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte("apiVersion: [this is: not valid"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, code := runCLI(t, env, wd, "--version")
		if code > 1 {
			t.Fatalf("--version exit %d with malformed config (expected 0 or 1)", code)
		}
	})

	t.Run("43_config_path_honours_spwn_home", func(t *testing.T) {
		// H43: distinct SPWN_HOME values produce distinct config.yaml files.
		t.Parallel()
		envA, homeA := freshEnv(t)
		envB, homeB := freshEnv(t)
		wd := t.TempDir()
		for _, env := range [][]string{envA, envB} {
			if _, _, code := runCLI(t, env, wd, "auth", "status"); code != 0 {
				t.Fatalf("auth status failed for %v", env)
			}
		}
		if homeA == homeB {
			t.Fatalf("SPWN_HOME fixtures collided: %s", homeA)
		}
		for _, h := range []string{homeA, homeB} {
			if _, err := os.Stat(filepath.Join(h, "config.yaml")); err != nil {
				t.Fatalf("config missing in %s: %v", h, err)
			}
		}
	})

	// --------------------------------------------------------------------
	// I — check rules
	// --------------------------------------------------------------------

	t.Run("44_check_flags_missing_description", func(t *testing.T) {
		// I44: an agent.yaml with no `description:` fails check.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		yamlPath := filepath.Join(wd, "spwn/agents/neo/agent.yaml")
		y := readFile(t, yamlPath)
		// Drop the existing description line(s).
		var out []string
		for _, line := range strings.Split(y, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "description:") {
				continue
			}
			out = append(out, line)
		}
		if err := os.WriteFile(yamlPath, []byte(strings.Join(out, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should fail on missing description")
		}
		if !strings.Contains(stdout+stderr, "description") {
			t.Fatalf("check output should mention description")
		}
	})

	t.Run("45_check_flags_missing_profile", func(t *testing.T) {
		// I45: deleting identity/profile.md surfaces a loud error.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if err := os.Remove(filepath.Join(wd, "spwn/agents/neo/identity/profile.md")); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should fail when profile.md is missing")
		}
		if !strings.Contains(stdout+stderr, "profile.md") {
			t.Fatalf("check output should mention profile.md")
		}
	})

	t.Run("46_one_agent_one_world", func(t *testing.T) {
		// I46: the same agent name in two worlds triggers the conflict rule.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		yamlPath := filepath.Join(wd, "spwn.yaml")
		y := readFile(t, yamlPath)
		if err := os.WriteFile(yamlPath, []byte(y+"  backup:\n    agents: [neo]\n    workspaces: [.]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should flag cross-world agent")
		}
		if !strings.Contains(stdout+stderr, "already deployed") {
			t.Fatalf("check output missing one-agent-one-world message")
		}
	})

	t.Run("47_reserved_world_name", func(t *testing.T) {
		// I47: `knowledge` cannot be used as a world name.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		// Hand-craft the project rather than init+mutate so the world
		// name takes effect unambiguously.
		mustWriteProject(t, wd, "acme", "knowledge", "neo")
		stdout, stderr, code := runCLI(t, env, wd, "check")
		if code == 0 {
			t.Fatalf("check should reject world named 'knowledge'")
		}
		if !strings.Contains(stdout+stderr, "knowledge") {
			t.Fatalf("check output should mention the reserved name")
		}
	})

	// --------------------------------------------------------------------
	// J — misc release-readiness
	// --------------------------------------------------------------------

	t.Run("48_version_prints_non_empty", func(t *testing.T) {
		// J48: `--version` prints a non-empty version string.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, _, code := runCLI(t, env, wd, "--version")
		if code != 0 {
			t.Fatalf("--version exit %d", code)
		}
		if strings.TrimSpace(stdout) == "" {
			t.Fatalf("version output was empty")
		}
	})

	t.Run("49_build_tree_only_passes", func(t *testing.T) {
		// J49: `build --tree-only` on a fresh init renders to disk.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "build", "--tree-only")
		if code != 0 {
			t.Fatalf("build --tree-only failed: code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
		}
	})

	t.Run("50_docs_match_help_4_layer", func(t *testing.T) {
		// J50: both `agent create --help` and the docs page say `4-layer Mind`.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, stderr, code := runCLI(t, env, wd, "agent", "create", "--help")
		if code != 0 {
			t.Fatalf("agent create --help failed: stderr=%s", stderr)
		}
		if !strings.Contains(stdout+stderr, "4-layer Mind") {
			t.Fatalf("agent create --help missing 4-layer Mind")
		}
		docPath := findDocPath(t, "docs/cli/spwn_agent_create.md")
		body := readFile(t, docPath)
		if !strings.Contains(body, "4-layer Mind") {
			t.Fatalf("doc %s missing 4-layer Mind", docPath)
		}
	})
}

// mustWriteProject writes a minimal but valid spwn project skeleton so
// I47 can point check at a world name ("knowledge") that init would
// never produce naturally.
func mustWriteProject(t *testing.T, root, projectName, worldName, agentName string) {
	t.Helper()
	manifest := fmt.Sprintf(`version: 2
name: %s

worlds:
  %s:
    agents: [%s]
    workspaces: ["."]
`, projectName, worldName, agentName)
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(root, "spwn/agents", agentName)
	if err := os.MkdirAll(filepath.Join(agentDir, "identity"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, layer := range []string{"skills", "playbooks", "journal"} {
		if err := os.MkdirAll(filepath.Join(agentDir, layer), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	agentYAML := fmt.Sprintf(`name: %s
description: Test agent for release readiness.

runtime:
  backend: "spwn:claude-code"

dependencies: []
`, agentName)
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(agentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENTS.md"), []byte("# "+agentName+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "identity/profile.md"), []byte("# Profile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn.lock"), []byte("# spwn.lock — DO NOT EDIT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// findDocPath walks up from the CWD looking for the given relative
// docs path. The test binary runs from apps/cli, so the walk has to
// climb two levels to reach the repo root.
func findDocPath(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate %s by walking up from %s", rel, wd)
	return ""
}
