# Preferences

station keeps its own choices in one file, `~/.config/station/preferences.json`, and reads a few environment variables once at start. Neither is a setting: [Settings](settings.md) is what station sets up on a machine ([ADR-015](../decisions/015-preferences-json.md)).

| Section          | Answers                                           |
| ---------------- | ------------------------------------------------- |
| preferences.json | The file, its groups, the family that writes each |
| Retired names    | What a file an older station wrote becomes        |
| Environment      | The variables station reads, all resolved once    |

## preferences.json

```json
{
    "name": "macbook",
    "macs": {
        "mac-mini": { "user": "jterrazz.agent", "key": "~/.ssh/id_ed25519", "lan": "192.168.1.20" }
    },
    "vm_user": "jterrazz",
    "share_list": ["session:*", "operator:claude:oauth-token"],
    "user_email": "admin@jterrazz.com",
    "user_name": "Jean-Baptiste Terrazzoni",
    "ports": { "max_per_vm": 10 },
    "off_here": ["dock-spacer"],
    "never_sleeps": true
}
```

The file is a set of groups. Each is owned by one family and written only by that family's calls (the rule `one-writer-per-store`); the order is the one `wire` lists them in, and a group left empty is not written. No secret is in it.

| Group                     | Family     | Holds                                                                                                                                                                                                                                                                                                                                      | Written by                                                  |
| ------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------- |
| `name`                    | `setup`    | this machine's name                                                                                                                                                                                                                                                                                                                        | `station setup`                                             |
| `macs`                    | `machines` | the other Macs this one drives: `user`, `key`, `lan` (the address it last answered with on your LAN, which `machines unlock` dials), and when it last answered with the VMs it held then, what it is (its hardware address among it, which `machines wake` sends to), and `tell_back` while a wake or a remind waits to tell of its answer | `machines add`, `machines forget`, every list               |
| `vm_user`                 | `machines` | the one login of every VM                                                                                                                                                                                                                                                                                                                  | `station setup`                                             |
| `share_list`              | `accounts` | the secrets whose accounts may be shared with a VM: a name, or a family as `session:*` ([Accounts](accounts.md#sharing-accounts))                                                                                                                                                                                                          | by hand; `settings apply claude-token` adds its name        |
| `user_email`, `user_name` | `setup`    | the owner's git identity                                                                                                                                                                                                                                                                                                                   | `station setup`                                             |
| `ports`                   | `ports`    | the policy of [shared ports](ports.md#the-policy): `range`, `max_per_vm`, `ignored_ports`, `min_port`, `https`, and `not_shared`, the ports stopped sharing as `<vm>:<port>`                                                                                                                                                               | by hand; `ports stop-sharing`, `ports share`, `ports https` |
| `off_here`                | `settings` | the settings turned Off here on this machine                                                                                                                                                                                                                                                                                               | `settings off`, `settings apply`                            |
| `never_sleeps`            | `awake`    | this Mac never sleeps: the engine's `awake` job holds it                                                                                                                                                                                                                                                                                   | `settings apply never-sleeps`, `settings off never-sleeps`  |

A key no group names is dropped at the next write and at `station setup`. A `v` field names the version of the shape the file was written at; absent, it is version 1, and a file a newer station wrote is refused rather than read. A file that does not parse stops every command that reads it with its line, never overwritten.

## Retired names

`station setup` on a Mac station v2 set up moves `~/.config/station/settings.json` to `preferences.json` once, byte for byte, only while no `preferences.json` is there, and says so. A group kept under a retired name is read under its new one and written under it at the next write: `here` is `name`, `hosts` is `macs`, `services` is `ports`. A v2 `lend_list` is read while no `share_list` is written.

## Environment

`wire` reads the environment once and passes typed options down; no other package reads a variable.

| Variable                                                               | Read for                                                                                                                                   |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `AGENT_PASSWORD`                                                       | pre-fills the `autologin` form                                                                                                             |
| `MISE_DATA_DIR`                                                        | where mise keeps its shims                                                                                                                 |
| `NO_COLOR`, `COLUMNS`                                                  | the output form and width                                                                                                                  |
| `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME` | the folders station uses, when absolute ([Disk](disk.md#the-folders))                                                                      |
| `STATION_TERMINAL`                                                     | this terminal's id, which `station use` keeps its machine under ([Machines](machines.md#this-terminals-machine))                           |
| `GOENV`                                                                | where `station setup` writes Go's env file; `off` writes none                                                                              |
| `TART_HOME`                                                            | set by the tart launcher, never by the owner ([Disk](disk.md#tart))                                                                        |
| `STATION_PUBLIC_HOST`                                                  | set inside a VM by its shell block: the public name its ports are shared at ([Ports](ports.md#profiles))                                   |
| `STATION_GITHUB_API_URL`                                               | the GitHub API origin, so a spec can point it at a stub                                                                                    |
| `STATION_SYSTEM_DIRS`, `STATION_MACHINE_KIND`                          | which system folders count, and the machine kind, for a spec                                                                               |
| `STATION_SYSTEM_ROOT`                                                  | the root `/Applications`, `/opt/homebrew`, `/usr/local`, `/Library` and a VM's `/proc` are read under, for a spec                          |
| `STATION_KERNEL_STUB`                                                  | a folder of recorded kernel tables a Mac's activity is read from, for a spec                                                               |
| `STATION_POWER_STUB`                                                   | a folder whose files are the IOKit reading and whose calls go to `$HOME/.calls`, for a spec                                                |
| `STATION_GUEST_SSH_PORT`                                               | the port the engine dials a VM's sshd on (default 22), for a spec's stub sshd                                                              |
| `STATION_EXECUTABLE`                                                   | the binary `setup` runs again detached, for a spec's stub                                                                                  |
| `STATION_WORLD`, `STATION_WORLD_SEED`, `STATION_WORLD_HYPERVISOR`      | the spec build's world file, its seed and its hypervisor ([Test](../guides/test.md#a-use-case-over-a-world)); the release build reads none |
| `STATION_XDG_DETACHED`                                                 | internal: `setup`'s detached layout step                                                                                                   |

## Related

- [Settings](settings.md): what station sets up, and Off here.
- [Footprint](footprint.md): every file station writes.
- [ADR-007](../decisions/007-files-live-in-the-xdg-folders.md): why the XDG folders.
