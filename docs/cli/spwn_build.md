---
title: "spwn build"
slug: "spwn-build"
---

## spwn build

Transpile the project and compile it into a Docker image

### Synopsis

Transpile the project with the target runtime (default: claude-code)
and compile the result into a derived Docker compile.

The image is FROM spwn-world:latest by default, with the transpiled
tree COPY'd to /world/. The resulting image carries the project's
name and the runtime name as Docker labels, so it's push-ready and
reproducible.

Pass --tree-only to stop after the transpile step and write the
generated file tree to --output (default: ./dist). No Docker
required, useful for previewing renderer output or authoring a
new runtime backend.

Use 'spwn up' to spawn a world from the current project. Use 'spwn
check --deep' to run the transpile dry-run as part of validation.

Examples:
  spwn build                                  # transpile + image, tag spwn-<project>:latest
  spwn build --tag spwn-myproj:v1
  spwn build --base spwn-world:2.1
  spwn build --runtime claude-code
  spwn build --world <name>                   # multi-world projects
  spwn build --no-cache
  spwn build --json
  spwn build --tree-only                      # transpile only, write to ./dist
  spwn build --tree-only --output ./preview
  spwn build --tree-only --dry-run            # list paths, touch nothing
  spwn build --tree-only --agent neo          # filter to one agent

```
spwn build [flags]
```

### Options

```
      --agent string     Compile only the named agent (tree-only mode)
      --base string      Base image to derive from (default: $SPWN_BASE_IMAGE, else spwn-world:latest)
      --dry-run          Print paths that would be written, don't touch disk (requires --tree-only)
      --force            Overwrite existing files in --output without prompting (requires --tree-only)
  -h, --help             help for build
      --json             Emit a machine-readable build report on stdout
      --no-cache         Disable Docker build cache
  -o, --output string    Output directory for --tree-only mode (default "dist")
      --runtime string   Target runtime. Defaults to the runtime declared in agent.yaml (fallback: claude-code)
      --tag string       Image tag (default: spwn-<project>:latest)
      --tree-only        Stop after the compile step; write the Tree to --output instead of building a Docker image
      --world string     World from spwn.yaml to build (required for multi-world projects)
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

