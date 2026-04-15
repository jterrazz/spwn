---
title: "spwn compile"
slug: "spwn-compile"
---

## spwn compile

Compile the project into a runtime-specific file tree

### Synopsis

Render the project through the claude-code runtime and materialise
the resulting Tree to disk.

Useful for previewing what spwn up would bake into its container,
debugging renderer output, and packaging for non-Docker runtimes.

  spwn compile                      # -> ./dist
  spwn compile --out ./preview
  spwn compile --dry-run            # list paths, touch nothing
  spwn compile --agent neo          # filter to one agent
  spwn compile --json               # machine-readable report

```
spwn compile [flags]
```

### Options

```
      --agent string     Compile only the named agent (filter the Tree)
      --dry-run          Print paths that would be written, don't touch disk
      --force            Overwrite existing files in --out without prompting
  -h, --help             help for compile
      --json             Emit a machine-readable build report on stdout
      --out string       Output directory for the compiled tree (default "dist")
      --runtime string   Target runtime (defaults to the runtime declared in agent.yaml, fallback: claude-code)
      --world string     World from spwn.yaml to compile (default: sole world)
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

