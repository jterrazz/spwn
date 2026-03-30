package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cli "spwn.sh/apps/cli"

	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gen-docs <output-dir>")
		os.Exit(1)
	}
	outDir := os.Args[1]

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Frontmatter prepender for Next.js pages.
	prepend := func(filename string) string {
		name := filepath.Base(filename)
		name = strings.TrimSuffix(name, ".md")
		title := strings.ReplaceAll(name, "_", " ")
		slug := strings.ReplaceAll(name, "_", "-")
		return fmt.Sprintf(`---
title: "%s"
slug: "%s"
---

`, title, slug)
	}

	// Link handler for cross-references between commands.
	link := func(name string) string {
		base := strings.TrimSuffix(name, ".md")
		slug := strings.ReplaceAll(base, "_", "-")
		return "/docs/cli/" + slug
	}

	cmd := cli.GetRootCmd()
	cmd.DisableAutoGenTag = true

	if err := doc.GenMarkdownTreeCustom(cmd, outDir, prepend, link); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Generated CLI docs in %s\n", outDir)
}
