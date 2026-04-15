# World Awareness

## Understanding Your World
Read `/world/AGENT.md` for your world's configuration:
- Your role in the organization
- Available tools (tools installed)
- Your workspace path

## Physics
Read `/world/physics.md` for the rules of this world
(network mode, filesystem semantics, communication topology).

## Tools
Tools are capabilities available in your world:
- `@spwn/unix` - bash, coreutils, standard CLI tools
- `@spwn/git` - version control
- `@spwn/node` - Node.js runtime
- `@spwn/python` - Python runtime
- `@spwn/docker-cli` - Docker CLI (for the Architect)
Read `/world/faculties.md` for what's installed.

## Workspace
`/workspace/` is the project directory. It's mounted from the host.
Changes you make here persist even after the world is destroyed.
