---
title: "spwn build"
slug: "spwn-build"
---

## spwn build

Compile the project and bake it into a Docker image

### Synopsis

Compile the project with the target runtime (default: claude-code)
and bake the result into a derived Docker image.

The image is FROM spwn-world:latest by default, with the compiled
tree COPY'd to /world/. The resulting image carries the project's
name and the runtime name as Docker labels, so it's push-ready and
reproducible.

Use 'spwn compile' for the compile step alone (no Docker required).
Use 'spwn up' to spawn a world from the current project. Use 'spwn
check --deep' to run the compile dry-run as part of validation.

Examples:
  spwn build                                  # default tag: spwn-<project>:latest
  spwn build --tag spwn-myproj:v1
  spwn build --base spwn-world:2.1
  spwn build --runtime claude-code
  spwn build --world <name>                   # multi-world projects
  spwn build --no-cache
  spwn build --json

```
spwn build [flags]
```

### Options

```
      --base string      Base image to derive from (default: $SPWN_BASE_IMAGE, else spwn-world:latest)
  -h, --help             help for build
      --json             Emit a machine-readable build report on stdout
      --no-cache         Disable Docker build cache
      --runtime string   Target runtime. Defaults to the runtime declared in agent.yaml (fallback: claude-code)
      --tag string       Image tag (default: spwn-<project>:latest)
      --world string     World from spwn.yaml to build (required for multi-world projects)
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

