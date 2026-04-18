# Physics of This World

## Laws
- Network: bridge (outbound access enabled)
- Filesystem is ephemeral except /workspaces and /mind

## Tools
/workspaces - project files, mounted from Host (read-write)
/mind - agent identity and memory (read-write)
/tmp - ephemeral scratch space

## Communication
Agents communicate via the inbox at /world/inbox/.
To send a message: write a JSON file to /world/inbox/{recipient}/.
To check messages: read files from /world/inbox/{your-name}/.

## Topology
/workspaces - project files, mounted from Host (read-write)
/mind - agent identity and memory (read-write)
/tmp - ephemeral scratch space
