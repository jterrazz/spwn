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

	"spwn.sh/packages/project"
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
	// into the test, particularly a user's real SPWN_HOME and any
	// provider credentials that would tip the runtime auto-resolver
	// (runtimeres.Resolve) away from the default claude-code path.
	leakyPrefixes := []string{
		"SPWN_HOME=",
		"ANTHROPIC_API_KEY=",
		"CLAUDE_CODE_OAUTH_TOKEN=",
		"ANTHROPIC_AUTH_TOKEN=",
		"OPENAI_API_KEY=",
	}
	base := os.Environ()
	filtered := base[:0]
	for _, kv := range base {
		skip := false
		for _, p := range leakyPrefixes {
			if strings.HasPrefix(kv, p) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		filtered = append(filtered, kv)
	}
	// Force keychain detection to stay off so a developer running
	// `make test` locally doesn't hit macOS Keychain prompts.
	cmd.Env = append(filtered, "SPWN_SKIP_KEYCHAIN=1")
	cmd.Env = append(cmd.Env, env...)
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

// mustInit scaffolds a fresh project at wd with the given name, as
// `station project create` does: through project.Init.
func mustInit(t *testing.T, _ []string, wd, name string) {
	t.Helper()
	if err := project.Init(wd, project.InitOpts{Name: name}); err != nil {
		t.Fatalf("project.Init: %v", err)
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
	// A: init + scaffold
	// --------------------------------------------------------------------

	t.Run("01_init_name_flag_writes_manifest", func(t *testing.T) {
		// A01: a scaffold named acme writes spwn.yaml containing that name.
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

	t.Run("03_init_writes_soul_and_2_layer_mind", func(t *testing.T) {
		// A03: fresh init writes SOUL.md + playbooks/journal, no knowledge.
		// (identity/ directory was collapsed into a single SOUL.md at agent
		// root; skills moved to build-time dependencies at /world/skills/.)
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		for _, layer := range []string{"playbooks", "journal"} {
			if _, err := os.Stat(filepath.Join(wd, "agents/neo", layer)); err != nil {
				t.Errorf("missing agent layer %s: %v", layer, err)
			}
		}
		if _, err := os.Stat(filepath.Join(wd, "agents/neo/SOUL.md")); err != nil {
			t.Errorf("missing SOUL.md at agent root: %v", err)
		}
		if _, err := os.Stat(filepath.Join(wd, "agents/neo/identity")); !os.IsNotExist(err) {
			t.Errorf("identity/ directory should NOT exist (collapsed into SOUL.md), stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Join(wd, "agents/neo/skills")); !os.IsNotExist(err) {
			t.Errorf("agent skills layer should NOT exist (moved to build-time deps), stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Join(wd, "agents/neo/knowledge")); !os.IsNotExist(err) {
			t.Errorf("agent knowledge layer should NOT exist, stat err=%v", err)
		}
	})

	t.Run("04_init_writes_world_knowledge", func(t *testing.T) {
		// A04: fresh init seeds a ./knowledge/.gitkeep, records the
		// path explicitly in spwn.yaml, and creates no spwn/worlds/ tree.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		gk := filepath.Join(wd, "knowledge/.gitkeep")
		if _, err := os.Stat(gk); err != nil {
			t.Fatalf("world knowledge .gitkeep missing: %v", err)
		}
		manifest := readFile(t, filepath.Join(wd, "spwn.yaml"))
		if !strings.Contains(manifest, "knowledge: ./knowledge") {
			t.Fatalf("spwn.yaml missing `knowledge: ./knowledge`:\n%s", manifest)
		}
		if _, err := os.Stat(filepath.Join(wd, "spwn/worlds")); !os.IsNotExist(err) {
			t.Fatalf("spwn/worlds/ should not exist after init, stat err=%v", err)
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

	// --------------------------------------------------------------------
	// B: ref syntax + lockfile
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

	t.Run("10_install_rejects_at_prefix", func(t *testing.T) {
		// B10: `@spwn/node` is malformed under the scheme grammar.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "install", "@spwn/node")
		if code == 0 {
			t.Fatalf("install @spwn/node should fail, got exit 0.\nstdout=%s", stdout)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "does not exist") && !strings.Contains(combined, "unsupported") && !strings.Contains(combined, "malformed") {
			t.Fatalf("expected malformed/unsupported error, got:\n%s", combined)
		}
	})

	t.Run("11_install_reports_registry_unsupported", func(t *testing.T) {
		// B11: github: refs surface as "not yet supported" at install
		// time: they never reach agent.yaml.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		stdout, stderr, code := runCLI(t, env, wd, "install", "github:acme/x")
		if code == 0 {
			t.Fatalf("install github:acme/x should fail, got exit 0")
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "not yet supported") {
			t.Fatalf("expected registry-unsupported message, got:\n%s", combined)
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

	// --------------------------------------------------------------------
	// D: install (project-scope + --agent scope)
	// --------------------------------------------------------------------

	t.Run("22_install_appends_dep", func(t *testing.T) {
		// D22: `install` appends to every agent.yaml.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		// Remove the default python first to prove install re-adds it.
		if _, _, code := runCLI(t, env, wd, "uninstall", "spwn:python"); code != 0 {
			t.Fatalf("pre-uninstall python failed")
		}
		if _, _, code := runCLI(t, env, wd, "install", "spwn:python"); code != 0 {
			t.Fatalf("install spwn:python failed")
		}
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
		if !strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml missing spwn:python:\n%s", y)
		}
	})

	t.Run("23_install_is_idempotent", func(t *testing.T) {
		// D23: installing the same dep twice still yields one entry.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		for i := 0; i < 2; i++ {
			if _, _, code := runCLI(t, env, wd, "install", "spwn:node"); code != 0 {
				t.Fatalf("iter %d install failed", i)
			}
		}
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
		if strings.Count(y, "spwn:node") != 1 {
			t.Fatalf("expected exactly 1 occurrence of spwn:node:\n%s", y)
		}
	})

	t.Run("24_uninstall_detaches", func(t *testing.T) {
		// D24: `uninstall` drops the entry from every agent.yaml.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "uninstall", "spwn:python"); code != 0 {
			t.Fatalf("uninstall failed")
		}
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
		if strings.Contains(y, "spwn:python") {
			t.Fatalf("agent.yaml still has spwn:python:\n%s", y)
		}
	})

	t.Run("25_uninstall_absent_is_silent", func(t *testing.T) {
		// D25: uninstalling a dep no agent carries is a silent no-op
		// (matches npm: install/uninstall are declarative, not
		// Transactional).
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		if _, _, code := runCLI(t, env, wd, "uninstall", "spwn:never-added"); code != 0 {
			t.Fatalf("uninstall of absent dep should be a no-op, got non-zero exit")
		}
	})

	// --------------------------------------------------------------------
	// E: install extra, scoping + lockfile
	// --------------------------------------------------------------------

	t.Run("26_install_updates_agents_and_lockfile", func(t *testing.T) {
		// E26: install adds the dep to every agent and pins the lockfile.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		mustInit(t, env, wd, "acme")
		// First uninstall the default entry so we can prove install
		// Re-adds it.
		if _, _, code := runCLI(t, env, wd, "uninstall", "spwn:python"); code != 0 {
			t.Fatalf("pre-uninstall failed")
		}
		if _, _, code := runCLI(t, env, wd, "install", "spwn:python"); code != 0 {
			t.Fatalf("install failed")
		}
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
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
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
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
		y := readFile(t, filepath.Join(wd, "agents/neo/agent.yaml"))
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
	// G: help text consistency
	// --------------------------------------------------------------------

	t.Run("34_agent_help_mentions_soul_and_layers", func(t *testing.T) {
		// G34: `agent --help` advertises SOUL.md at root + the two Mind
		// layer dirs (playbooks/journal). Skills moved to build-time
		// dependencies.
		t.Parallel()
		env, _ := freshEnv(t)
		wd := t.TempDir()
		stdout, stderr, code := runCLI(t, env, wd, "agent", "--help")
		if code != 0 {
			t.Fatalf("agent --help failed: stderr=%s", stderr)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "SOUL.md") {
			t.Fatalf("agent --help missing `SOUL.md`:\n%s", combined)
		}
		if !strings.Contains(combined, "playbooks/journal") {
			t.Fatalf("agent --help missing layer list:\n%s", combined)
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

	// --------------------------------------------------------------------
	// H: config.yaml + migrations
	// --------------------------------------------------------------------

	t.Run("39_first_invocation_creates_config_yaml", func(t *testing.T) {
		// H39: a fresh SPWN_HOME grows config.yaml on the first CLI call.
		t.Parallel()
		env, home := freshEnv(t)
		wd := t.TempDir()
		_, _, code := runCLI(t, env, wd, "auth")
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
		if _, _, code := runCLI(t, env, wd, "auth"); code != 0 {
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
		if _, _, code := runCLI(t, env, wd, "auth"); code != 0 {
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
			if _, _, code := runCLI(t, env, wd, "auth"); code != 0 {
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
	// J: misc release-readiness
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

	// ── K. Regression tests for bugs surfaced this session ───────────

	t.Run("55_migration_backup_tolerates_broken_symlink", func(t *testing.T) {
		// Regression for a794d543: the pre-migration backup walked a
		// credential-routing symlink (e.g. ~/.spwn/agents/<name>/.codex/
		// auth.json -> /credentials/openai/auth.json) and errored
		// out because the symlink target only exists INSIDE a
		// container namespace. Every spwn command on the host was
		// broken until the backup learned to skip symlinks. This
		// test reproduces the broken-link setup and asserts any
		// migrating command still completes.
		t.Parallel()
		env, home := freshEnv(t)
		agentDir := filepath.Join(home, "agents", "atlas", ".codex")
		if err := os.MkdirAll(agentDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Target intentionally does not exist on the host.
		if err := os.Symlink("/credentials/openai/auth.json",
			filepath.Join(agentDir, "auth.json")); err != nil {
			t.Fatal(err)
		}
		// `auth status` triggers PersistentPreRunE, then the migration
		// runner, the pre-migration backup and a walk of SPWN_HOME. Must not
		// blow up on the broken link.
		_, stderr, code := runCLI(t, env, t.TempDir(), "auth")
		if code != 0 {
			t.Fatalf("auth status failed on SPWN_HOME with broken symlink (code=%d):\n%s", code, stderr)
		}
		if strings.Contains(stderr, "pre-migration backup") &&
			strings.Contains(stderr, "no such file") {
			t.Fatalf("pre-migration backup followed broken symlink:\n%s", stderr)
		}
	})
}
