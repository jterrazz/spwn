# ADR-012: Docker-out-of-Docker (DooD) over Docker-in-Docker (DinD)

**Status:** Accepted
**Date:** 2026-04-01

## Context

Spwn's Architect daemon runs in a Docker container and needs to create and manage World containers. Two approaches exist:

1. **Docker-in-Docker (DinD)**: Run a full Docker daemon inside the Architect container. World containers are nested inside the Architect.
2. **Docker-out-of-Docker (DooD)**: Mount the host's Docker socket (`/var/run/docker.sock`) into the Architect container. World containers are created as siblings on the host's Docker daemon.

## Decision

Use **Docker-out-of-Docker (DooD)**. The Architect container mounts `/var/run/docker.sock` and creates sibling containers on the host's Docker daemon.

### How It Works

```
Host machine
└── Docker daemon (one daemon, controls everything)
    ├── Architect container (has /var/run/docker.sock mounted)
    │   ├── spwn binary + core libraries
    │   ├── ~/.spwn/ (mounted from host)
    │   └── Connected: Telegram, Slack, Discord, CLI
    ├── World: w-jupiter-02572 (sibling, created BY Architect)
    ├── World: w-callisto-38907 (sibling, created BY Architect)
    └── Observatory container (sibling, created BY Architect)
```

The Architect container runs with:

```bash
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ~/.spwn:/root/.spwn \
  --network spwn-net \
  spwn/architect
```

### Two Deployment Modes

The same code and same Backend port work in both modes:

- **LOCAL MODE**: The `spwn` CLI runs directly on the host, talks to `docker.sock` natively, and creates World containers. No Architect container needed. Fast, zero overhead.
- **HOSTED MODE**: The Architect container runs always-on, talks to the mounted `docker.sock`, and creates World containers. Supports multi-channel input (Telegram, Slack, Discord).

### Inter-Container Communication

All containers (Architect, Worlds, Observatory) join the `spwn-net` Docker bridge network. This enables:

- Architect → World communication (task routing, lifecycle management)
- World → World communication (if needed for multi-world orchestration)
- Observatory → Architect event streaming (via WebSocket)

### Shared State

The `~/.spwn/` directory is mounted from the host into the Architect container. This ensures:

- Agent profiles, world configs, and state persist across container restarts
- The same config is accessible in both local and hosted modes
- Config sync (git-based) works identically in both modes

## Consequences

### Positive

- **Performance**: No nested Docker daemon overhead. World containers run at native speed with direct access to host resources (cgroups, storage drivers).
- **Simplicity**: One Docker daemon to manage. Standard Docker tooling (`docker ps`, `docker logs`) shows all containers—Architect, Worlds, Observatory—as peers.
- **Storage efficiency**: No copy-on-copy layering. Image layers are shared between sibling containers on the same daemon.
- **Industry standard**: DooD is the established pattern for CI/CD systems (Jenkins, GitLab CI) and container orchestrators. Well-documented, well-understood.
- **Debuggability**: `docker ps` on the host shows everything. No need to exec into the Architect container to inspect World containers.
- **Two modes from one codebase**: The Backend port talks to `docker.sock` regardless of whether it's the CLI or the Architect calling it.

### Negative

- **Socket access = root-equivalent**: The Architect container can create any container on the host. Mitigated by the fact that the user explicitly opts in by running `spwn architect start`, and the Architect is trusted code.
- **No nested isolation**: World containers share the host's Docker daemon. A malicious World container with Docker socket access could affect siblings. Mitigated by never mounting `docker.sock` into World containers—only the Architect has socket access.

### Why Not DinD

- **Complexity**: DinD requires `--privileged` mode or complex security configurations. It runs a full Docker daemon inside a container, doubling resource usage.
- **Performance overhead**: Nested storage drivers (overlay-on-overlay) cause significant I/O degradation.
- **Caching issues**: Image layers cannot be shared between the inner and outer Docker daemons, leading to redundant downloads and storage.
- **Debugging difficulty**: World containers are invisible to the host's `docker ps`. Operators must exec into the Architect to inspect them.
- **Fragility**: Inner Docker daemons can conflict with the outer daemon on cgroups, networking, and storage.
