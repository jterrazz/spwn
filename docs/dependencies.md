# Dependencies

How spwn projects declare, install, and resolve the tools, skills, and capabilities their agents need.

## The model

External dependencies live in **`dependencies:`** — a flat list in `spwn.yaml` (project-wide) and optionally in each `agent.yaml` (agent-specific additions). Local authored blocks live in **typed directories** under `spwn/`.

```yaml
# spwn.yaml — project manifest
version: 2
name: acme-api

dependencies:                      # project-wide — every agent inherits these
  - "@spwn/unix"
  - "@spwn/git"
  - "@spwn/python"
  - "@spwn/mempalace"

worlds:
  matrix:
    agents: [neo]
    workspaces: [.]
```

```yaml
# spwn/agents/neo/agent.yaml — agent composition
name: neo
runtime:
  backend: "@spwn/claude-code"

dependencies:                      # agent-specific additions (on top of project deps)
  - "@spwn/qmd"

skills:                            # local skills this agent uses
  - paper-reading
  - code-review

tools:                             # local tool definitions
  - ffmpeg

hooks:                             # lifecycle hooks
  - pre-spawn
```

## Reference kinds

Every ref in `dependencies:` is one of three kinds:

| Kind | Syntax | Resolved from |
|------|--------|--------------|
| **Builtin** | `@spwn/<name>` | Catalog compiled into the spwn binary |
| **GitHub** | `github.com/<owner>/<repo>` | Git clone + git tags as versions (planned) |
| **Local** | `<bare-name>` | `spwn/tools/<name>/` directory |

Bare names in typed sections resolve from their matching directory:

| Section | Resolves from |
|---------|--------------|
| `skills:` | `spwn/skills/<name>.md` |
| `tools:` | `spwn/tools/<name>/` (with `spwn.yaml`) |
| `hooks:` | `spwn/hooks/<name>.sh` |

## Local hierarchy

```
my-project/
├── spwn.yaml              # project manifest + dependencies
├── spwn.lock              # pinned versions (DO NOT EDIT)
├── spwn/
│   ├── agents/
│   │   └── neo/
│   │       ├── agent.yaml     # agent dependencies + local block references
│   │       ├── AGENTS.md      # agent prompt (provider-neutral)
│   │       ├── identity/      # who the agent is
│   │       ├── skills/        # agent-scoped skills (only this agent)
│   │       ├── knowledge/     # learned facts
│   │       ├── playbooks/     # promoted workflows
│   │       └── journal/       # session history
│   ├── skills/                # project-wide shared skills
│   │   ├── paper-reading.md
│   │   └── code-review.md
│   ├── tools/                 # local tool definitions
│   │   └── ffmpeg/
│   │       └── spwn.yaml
│   └── hooks/                 # lifecycle hook scripts
│       └── pre-spawn.sh
└── .spwn/                     # gitignored local state
```

## Lock file

`spwn.lock` is a line-oriented text file — one entry per dependency. Human-readable, trivially diffable in PRs.

```
# spwn.lock — DO NOT EDIT
@spwn/git latest builtin
@spwn/mempalace latest builtin
@spwn/python latest builtin
@spwn/unix latest builtin
```

Each line: `<ref> <version> <source>`. Managed by `spwn install` / `spwn uninstall`. Commit this file.

## CLI

```bash
# Install / remove external dependencies
spwn install @spwn/python                    # add to every agent + pin in spwn.lock
spwn install github.com/jterrazz/skills      # (planned) install from GitHub
spwn uninstall @spwn/python                  # remove from agents + lockfile

# Author local blocks
spwn skill new paper-reading                 # create spwn/skills/paper-reading.md
spwn skill edit paper-reading                # open in $EDITOR
spwn skill show paper-reading                # display
spwn skill rm paper-reading                  # delete
spwn skill ls                                # list all skills (local + from dependencies)
```

## Resolution flow

When `spwn up` or `spwn build` runs:

1. **Collect dependencies**: merge project-level `spwn.yaml#dependencies` + each agent's `agent.yaml#dependencies`
2. **Resolve transitive**: for each dependency, recursively expand its `dependencies:` field (from `spwn.yaml`)
3. **Topological sort**: Kahn's algorithm orders dependencies so dependencies come before dependents
4. **Deduplicate**: same ref from multiple sources appears once
5. **Build image**: generate Dockerfile with install steps from each resolved dependency
6. **Compile tree**: render per-agent `CLAUDE.md`, skills, world files into the container

Local blocks (skills, tools, hooks) are discovered from the `spwn/` directories and compiled into the tree alongside dependency-provided content.

## Inheritance

Agents inherit all project-level dependencies automatically. An agent's `dependencies:` section adds to the project pool — it cannot remove from it.

```yaml
# spwn.yaml
dependencies: ["@spwn/unix", "@spwn/git"]     # every agent gets these

# agent.yaml for neo
dependencies: ["@spwn/python"]                # neo also gets python
# neo's resolved dependencies: @spwn/unix + @spwn/git + @spwn/python
```

## Version pinning

Catalog dependencies (`@spwn/*`) are compiled into the binary — their version is the spwn CLI version. GitHub dependencies (planned) will use git tags with minimum version selection:

```yaml
dependencies:
  - github.com/acme/tools           # latest tag
  - github.com/acme/tools@1.2       # minimum v1.2.x
  - github.com/acme/tools@1.2.3     # exact pin
```

The lock file records the resolved version so builds are reproducible.

## Publishing a dependency

A publishable dependency is a spwn project without `worlds:`. Push it to GitHub, tag a release, and anyone can install it:

```yaml
# spwn.yaml in a dependency repo
name: my-research-tools
version: "1.0.0"
description: "Tools for scientific agents"

# No worlds: — this is a distributable dependency, not a project
```

```
my-dep-repo/
├── spwn.yaml
└── spwn/
    ├── tools/
    │   └── jupyter/
    │       └── spwn.yaml
    ├── skills/
    │   └── scientific-method.md
    └── hooks/
        └── post-spawn.sh
```

Install it: `spwn install github.com/you/my-research-tools`
