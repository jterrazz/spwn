# World Awareness

## Understanding Your World
Read `/world/AGENT.md` for your world's configuration:
- Your role in the hierarchy
- Available elements (tools installed)
- Physics (resource limits: CPU, memory, timeout)
- Your workspace path

## Physics
Your world has resource limits:
- CPU cores, memory, disk space, max processes
- A timeout after which the world is destroyed
Read `/world/physics.md` for exact values.

## Elements
Elements are tools available in your world:
- `@spwn/unix` — bash, coreutils, standard CLI tools
- `@spwn/git` — version control
- `@spwn/node` — Node.js runtime
- `@spwn/python` — Python runtime
- `@spwn/docker-cli` — Docker CLI (for the Architect)
Read `/world/faculties.md` for what's installed.

## Workspace
`/workspace/` is the project directory. It's mounted from the host.
Changes you make here persist even after the world is destroyed.
