package catalog

import (
	"io/fs"
	"strings"
	"testing"

	ib "spwn.sh/core/imagebuilder"
)

func TestAllTools_ValidName(t *testing.T) {
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			if !strings.HasPrefix(tool.Name(), "@") {
				t.Errorf("tool name %q must start with @", tool.Name())
			}
			if tool.Name() == "@" {
				t.Error("tool name must not be just @")
			}
		})
	}
}

func TestAllTools_ValidKind(t *testing.T) {
	validKinds := map[ib.Kind]bool{
		ib.KindRuntime:  true,
		ib.KindTool:     true,
		ib.KindSDK:      true,
		ib.KindPlatform: true,
	}
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			if !validKinds[tool.Kind()] {
				t.Errorf("invalid kind %q", tool.Kind())
			}
		})
	}
}

func TestAllTools_VersionNotEmpty(t *testing.T) {
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			if tool.Version() == "" {
				t.Error("version must not be empty")
			}
		})
	}
}

func TestAllTools_VerifyNotEmpty(t *testing.T) {
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			if len(tool.Verify()) == 0 {
				t.Errorf("%s must have at least one verify command", tool.Name())
			}
		})
	}
}

func TestAllTools_InstallSpecNonEmpty(t *testing.T) {
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			spec := tool.Install()
			hasContent := len(spec.Packages) > 0 || len(spec.Commands) > 0 || len(spec.Files) > 0
			if !hasContent {
				t.Errorf("%s install spec must have packages, commands, or files", tool.Name())
			}
		})
	}
}

func TestAllTools_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range All {
		if seen[tool.Name()] {
			t.Errorf("duplicate tool name: %s", tool.Name())
		}
		seen[tool.Name()] = true
	}
}

func TestAllTools_DependenciesExist(t *testing.T) {
	reg := ib.NewRegistry()
	RegisterDefaults(reg)

	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			for _, dep := range tool.Dependencies() {
				if reg.Get(dep) == nil {
					t.Errorf("%s depends on %s, which is not registered", tool.Name(), dep)
				}
			}
		})
	}
}

func TestAllTools_NoDependencyCycles(t *testing.T) {
	reg := ib.NewRegistry()
	RegisterDefaults(reg)

	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			_, err := reg.Resolve([]string{tool.Name()})
			if err != nil {
				t.Errorf("resolve %s failed: %v", tool.Name(), err)
			}
		})
	}
}

func TestAllTools_NoDependOnSelf(t *testing.T) {
	for _, tool := range All {
		t.Run(tool.Name(), func(t *testing.T) {
			for _, dep := range tool.Dependencies() {
				if dep == tool.Name() {
					t.Errorf("%s depends on itself", tool.Name())
				}
			}
		})
	}
}

func TestAllTools_SkillsHaveSkillMD(t *testing.T) {
	for _, tool := range All {
		s := tool.Skills()
		if s == nil {
			continue
		}
		t.Run(tool.Name(), func(t *testing.T) {
			_, err := fs.ReadFile(s, "SKILL.md")
			if err != nil {
				t.Errorf("%s has Skills() but no SKILL.md: %v", tool.Name(), err)
			}
		})
	}
}

func TestRegisterDefaults_AllRegistered(t *testing.T) {
	reg := ib.NewRegistry()
	RegisterDefaults(reg)

	for _, tool := range All {
		if reg.Get(tool.Name()) == nil {
			t.Errorf("tool %s not found after RegisterDefaults", tool.Name())
		}
	}
}

func TestResolve_FullWorldStack(t *testing.T) {
	reg := ib.NewRegistry()
	RegisterDefaults(reg)

	tools, err := reg.Resolve([]string{"@unix", "@git", "@node", "@claude-code", "@spwn", "@qmd"})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	// @node must come before @claude-code and @qmd
	idx := make(map[string]int)
	for i, tool := range tools {
		idx[tool.Name()] = i
	}

	if idx["@node"] >= idx["@claude-code"] {
		t.Error("@node must come before @claude-code")
	}
	if idx["@node"] >= idx["@qmd"] {
		t.Error("@node must come before @qmd")
	}
}
