package catalog_test

import (
	"spwn.sh/packages/dependency/builtin"
	"strings"
	"testing"


	"spwn.sh/packages/project"
	runtimespkg "spwn.sh/packages/runtimes"
	"spwn.sh/packages/transpile"
	_ "spwn.sh/packages/runtimes/claudecode" // register the claude-code compile renderer
	"spwn.sh/packages/transpile/source"
)

// TestCatalog_EveryGalleryEntryBuilds exercises the check-to-build
// pipeline against every gallery-eligible catalog entry. For each
// slug it:
//
//  1. Installs the entry into a fresh temp dir (what `spwn init`
//     does).
//  2. Loads the installed project via `project.Find` (same walker
//     the CLI uses).
//  3. Runs `project.Validate` (the rule engine behind `station project plan`).
//  4. Loads the project source and calls `transpile.Compile` (the
//     pure half of `spwn build --tree-only`, no Docker).
//
// If any of those steps fails, the catalog entry is broken at
// compile time and would fail for any user who ran `spwn init
// <slug> && station project plan`.
func TestCatalog_EveryGalleryEntryBuilds(t *testing.T) {
	slugs := builtin.GallerySlugs()
	if len(slugs) == 0 {
		t.Fatal("no gallery entries found: ShippedSlugs returned empty")
	}

	adapters := runtimespkg.All()
	builtins := make([]string, 0, len(builtin.BuiltinTools())+len(adapters))
	for _, tool := range builtin.BuiltinTools() {
		builtins = append(builtins, tool.Name())
	}
	supportedRuntimes := make([]string, 0, 2*len(adapters))
	for _, a := range adapters {
		if a.Tool != nil {
			builtins = append(builtins, a.Tool.Name())
		}
		supportedRuntimes = append(supportedRuntimes, a.Name)
		if a.Tool != nil {
			supportedRuntimes = append(supportedRuntimes, a.Tool.Name())
		}
	}

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			base := t.TempDir()

			// Step 1: install.
			if _, err := builtin.Install(slug, base); err != nil {
				t.Fatalf("Install %q: %v", slug, err)
			}

			// Step 2: find + load as a project.
			p, err := project.Find(base)
			if err != nil {
				t.Fatalf("project.Find: %v", err)
			}
			if p == nil {
				t.Fatal("project.Find returned nil, expected a project after Install")
			}

			// Step 3: validate (what `station project plan` runs).
			issues := project.Validate(p, project.ValidateOpts{
				BuiltinTools:      builtins,
				SupportedRuntimes: supportedRuntimes,
			})
			var errorIssues []string
			for _, iss := range issues {
				if iss.Level == project.LevelError {
					errorIssues = append(errorIssues, iss.Path+": "+iss.Message)
				}
			}
			if len(errorIssues) > 0 {
				t.Fatalf("validate surfaced %d error(s):\n  %s", len(errorIssues), strings.Join(errorIssues, "\n  "))
			}

			// Step 4: compile to tree (what `spwn build --tree-only`
			// does, minus the WriteTo).
			src, err := source.Load(p.Root)
			if err != nil {
				t.Fatalf("source.Load: %v", err)
			}
			in, err := source.ToCompileInput(src, "")
			if err != nil {
				t.Fatalf("source.ToCompileInput: %v", err)
			}
			tree, err := transpile.Compile("claude-code", in)
			if err != nil {
				t.Fatalf("transpile.Compile: %v", err)
			}
			if tree == nil || len(tree.Paths()) == 0 {
				t.Fatal("compile produced an empty tree")
			}
		})
	}
}
