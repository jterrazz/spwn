---
title: "spwn architect start"
slug: "spwn-architect-start"
---

## spwn architect start

Start the Architect daemon

### Synopsis

Start the Architect daemon in a Docker container.

The Architect runs the spwn binary inside a long-lived container with the
host's Docker socket mounted (DooD — Docker-outside-of-Docker), allowing it
to create and manage world containers as siblings.

The container mounts:
  /var/run/docker.sock    Docker daemon access (sibling containers, not nested)
  ~/.spwn/                Shared configuration and state

The Architect's identity is defined in /world/ARCHITECT.md inside the container,
which describes its capabilities and role as the always-on world builder.

```
spwn architect start [flags]
```

### Options

```
  -h, --help   help for start
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn architect](./spwn_architect.md)	 - Your always-on world builder

