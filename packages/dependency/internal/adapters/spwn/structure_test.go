package spwn

import (
	"io/fs"
	"testing"
)

// TestShippedSlugsMatchEmbed asserts every gallery-eligible entry
// (one with a `worlds:` section in spwn.yaml) is reachable via
// ShippedSlugs(), and vice-versa. Dependency-shaped entries (no
// worlds:) live in the same embed FS but stay out of the gallery.
//
// Lives here (inside the adapter, white-box) so it can walk the
// embed directly and catch contract drift before anything else
// reaches for the catalog. Black-box facade tests live in
// tests/_catalog/.
func TestShippedSlugsMatchEmbed(t *testing.T) {
	embed := EmbedFS()
	entries, err := fs.ReadDir(embed, ".")
	if err != nil {
		t.Fatalf("read embed root: %v", err)
	}

	gallerySlugs := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := Get(e.Name()); err == nil {
			gallerySlugs[e.Name()] = true
		}
	}

	canonical := make(map[string]bool)
	for _, s := range ShippedSlugs() {
		canonical[s] = true
	}

	for slug := range canonical {
		if !gallerySlugs[slug] {
			t.Errorf("ShippedSlugs lists %q but its spwn.yaml has no worlds: section", slug)
		}
	}
	for slug := range gallerySlugs {
		if !canonical[slug] {
			t.Errorf("embedded gallery entry %q is missing from ShippedSlugs", slug)
		}
	}
}

// TestShippedSlugsStructure asserts every gallery entry ships the
// minimum filesystem contract that Install and Get depend on:
//
//	<slug>/spwn.yaml
//	<slug>/spwn.lock
//	<slug>/agents/<at-least-one-dir>/SOUL.md
//	<slug>/agents/<at-least-one-dir>/agent.yaml
//
// Without these, the binary ships but misbehaves at runtime.
func TestShippedSlugsStructure(t *testing.T) {
	embed := EmbedFS()
	for _, slug := range ShippedSlugs() {
		t.Run(slug, func(t *testing.T) {
			for _, p := range []string{slug + "/spwn.yaml", slug + "/spwn.lock"} {
				if _, err := fs.Stat(embed, p); err != nil {
					t.Errorf("missing %s: %v", p, err)
				}
			}

			agentEntries, err := fs.ReadDir(embed, slug+"/agents")
			if err != nil {
				t.Errorf("read %s/agents: %v", slug, err)
				return
			}
			hasAgent := false
			for _, e := range agentEntries {
				if !e.IsDir() {
					continue
				}
				hasAgent = true
				profilePath := slug + "/agents/" + e.Name() + "/SOUL.md"
				if _, err := fs.Stat(embed, profilePath); err != nil {
					t.Errorf("%s: agent %q missing SOUL.md", slug, e.Name())
				}
				agentYAML := slug + "/agents/" + e.Name() + "/agent.yaml"
				if _, err := fs.Stat(embed, agentYAML); err != nil {
					t.Errorf("%s: agent %q missing agent.yaml", slug, e.Name())
				}
			}
			if !hasAgent {
				t.Errorf("%s: no agent directory under agents/", slug)
			}
		})
	}
}
