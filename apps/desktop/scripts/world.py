"""The world build of station and the sandbox one of its doors runs in, shared by the recorder and the bridge.

Run alone, it prints the path of the world build of this tree, made by `go build -tags world` once per state of
the Go sources and kept as <STATION_BENCH or the temp folder>/worlds/station-world-<hash>, then reused by every
copy of the repository whose sources hash the same. In a Linux VM only, as the door asks the spec build's
environment: HOME, PATH and every station path in a temporary folder, never the user's.

WORLDS is the one list of the worlds the window records, each a world of specs/usecases/worlds/ and what a page
needs that it lacks: machinefiles on This Mac, the fake binaries of a fixture of specs/usecases/__fixtures__/, and
edits of its machines (a sign-in in progress, a program signed out or broken, an accounts switch the fake reads in
apps/engine/world/accounts.go, as the presence check a grant asks, a failed sign-in or a failing background task
dropped); a world of WORLDS may be a world of specs/usecases/worlds/ under another name, seeded otherwise. A world no
entry names is served as specs/usecases/worlds/ holds it.

Every folder a sandbox or a seed makes is removed when the process exits, by its end, an exception or SIGTERM.
"""

import atexit
import hashlib
import json
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[3]

# hazard: SIGTERM skips atexit unless it is turned into an exit, and only the main thread may set its handler, so it
# hazard: is set on import: record.py makes its temp folders from worker threads.
if signal.getsignal(signal.SIGTERM) is signal.SIG_DFL:
    signal.signal(signal.SIGTERM, lambda *_: sys.exit(143))


def temporary(prefix):
    """A temp folder removed at exit, by its end, an exception or SIGTERM."""
    folder = pathlib.Path(tempfile.mkdtemp(prefix=prefix))
    atexit.register(shutil.rmtree, folder, True)
    return folder


def build_digest():
    """The hash of what the world build is built from, path and bytes of each file in sorted order: the Go modules,
    every Go source of apps/engine and every file of catalog/."""
    digest = hashlib.sha256()
    modules = [ROOT / "go.work", *sorted(ROOT.glob("go.work.sum"))]
    for module in ("apps/engine", "catalog"):
        modules += [ROOT / module / "go.mod", *sorted((ROOT / module).glob("go.sum"))]
    sources = [*modules, *sorted((ROOT / "apps" / "engine").rglob("*.go")), *sorted((ROOT / "catalog").rglob("*"))]
    sources = [source for source in sources if source.is_file()]
    for source in sources:
        digest.update(str(source.relative_to(ROOT)).encode())
        digest.update(source.read_bytes())
    return digest.hexdigest()


def world_binary():
    kept = pathlib.Path(os.environ.get("STATION_BENCH") or tempfile.gettempdir()) / "worlds"
    kept.mkdir(parents=True, exist_ok=True)
    binary = kept / f"station-world-{build_digest()[:16]}"
    if binary.exists():
        return binary
    building = binary.with_suffix(f".building-{os.getpid()}")
    subprocess.run(["go", "build", "-tags", "world", "-o", str(building), "./apps/engine/cmd/station"], cwd=ROOT, check=True)
    building.rename(binary)
    return binary


def sandbox(seed, fixture=None):
    """A fresh HOME for one door over a copy of the world `seed`, with the fake binaries of `fixture` on its PATH."""
    home = temporary("window-world-")
    for sub in ("bin", "root"):
        (home / sub).mkdir()
    if fixture:
        shutil.copytree(ROOT / "specs" / "usecases" / "__fixtures__" / fixture / "bin", home / "bin", dirs_exist_ok=True)
    env = {
        "HOME": str(home), "PATH": str(home / "bin"), "NO_COLOR": "1", "COLUMNS": "80", "TZ": "UTC",
        "STATION_GITHUB_API_URL": "http://127.0.0.1:9", "STATION_MACHINE_KIND": "mac",
        "STATION_KERNEL_STUB": str(home / "kernel"), "STATION_POWER_STUB": str(home / "power"),
        "STATION_SYSTEM_DIRS": "", "STATION_SYSTEM_ROOT": str(home / "root"),
        "STATION_WORLD": str(home / "world.json"), "STATION_WORLD_SEED": str(seed),
        "STATION_TERMINAL": "spec-terminal",
    }
    return home, env


BASE = """machinefile: station.machinefile/v1
name: base
description: What every machine of the owner carries
apps: [jq]
"""


def machine(name, apps, settings=""):
    return {
        "machinefiles/base.machine.yaml": BASE,
        f"machinefiles/{name}.machine.yaml": "machinefile: station.machinefile/v1\n"
        f"name: {name}\ndescription: The owner's Mac\nmachine: {{ kind: computer, system: {{ os: macos }} }}\n"
        f"extends: [base]\napps: [{', '.join(apps)}]\n" + settings,
    }


LOCK = """lock: station.lock/v1
machinefile: macbook-pro
apps:
    - name: bat
      method: brew
      version: 0.25.0
    - name: homebrew
      method: script
      version: 5.0.1
    - name: jq
      method: brew
      version: 1.8.1
    - name: lazygit
      method: brew
      version: 0.55.1
"""


def applied(apps):
    return {**machine("macbook-pro", apps), "machinefiles/macbook-pro.lock": LOCK}


EXPIRES = "2026-11-04T11:00:00Z"


def exchange(name, id, program, kind, site, asking=False):
    """A sign-in in progress on a machine, so an accounts.open or accounts.answer of `id` has one to reach."""
    def edit(content):
        held = content["macs"][name].setdefault("accounts", {}).setdefault("exchanges", {})
        held[id] = {"program": program, "kind": kind, "site": site, "expires": EXPIRES, **({"asking": True} if asking else {})}
    return edit


def signed_out(name, program):
    def edit(content):
        content["macs"][name].get("accounts", {}).get("programs", {}).pop(program, None)
    return edit


def broken(name, program, who, reason):
    """A program of a machine whose sign-in no longer works, the engine's Needs you line of it follows."""
    def edit(content):
        content["macs"][name].setdefault("accounts", {}).setdefault("programs", {})[program] = {
            "Program": program, "State": "broken", "Who": who, "How": "own", "Reason": reason,
        }
    return edit


def switched(name, value):
    def edit(content):
        content["macs"][content["this_mac"]].setdefault("accounts", {})[name] = value
    return edit


def dropped(name, *path):
    """What a machine of the world holds at `path` (a failed sign-in, a failing background task), gone."""
    def edit(content):
        held = content["macs"][name]
        for key in path[:-1]:
            held = held.get(key, {})
        held.pop(path[-1], None)
    return edit


def leftover(name, vm, size):
    """A stopped disk station cloned that no machine record names, as a machine delete that did not finish leaves it."""
    def edit(content):
        held = content["macs"][name]
        held["vms"][vm] = {"disk": True, "born": 901, "size": {"CPUs": 0, "Memory": 0, "Disk": size}}
        held["cloned"] = {**held.get("cloned", {}), vm: 901}
    return edit


def holds_file(name, path, size):
    def edit(content):
        content["macs"][name]["disk"]["files"][path] = size
    return edit


def started(name, vm):
    def edit(content):
        content["macs"][name]["vms"][vm]["pid"] = 4711
    return edit


def unreadable(name, vm):
    def edit(content):
        content["macs"][name]["vms"][vm]["unreadable"] = True
    return edit


def running(name, *processes):
    def edit(content):
        content["macs"][name]["disk"]["running"] = list(processes)
    return edit


def listens(name, vm, port, process):
    def edit(content):
        content["macs"][name]["vms"][vm].setdefault("listening", []).append(
            {"Port": port, "Bind": "0.0.0.0", "Process": process, "PID": 3009})
    return edit


def kept_local(name, vm, port):
    """A share its owner chose `local`, as `ports share --to local` leaves it: reserved so, and on no tailnet share."""
    def edit(content):
        held = content["macs"][name]
        reserved = next(r for r in held["stores"]["reservations"] if r["vm"] == vm and r["guest_port"] == port)
        reserved.update({"by": "owner", "chosen": "local", "to": "local"})
        held["tailnet"]["shares"] = [s for s in held["tailnet"]["shares"] if s["Reserved"] != reserved["host_port"]]
    return edit


def plain(name):
    return (name, None, {}, [])


STATION_VM = "~/.local/share/station/vms/vms/linux-1/disk.img"


WORLDS = {
    **{name: plain(name) for name in (
        "bare-mac", "creating", "disk-low", "engine-stopped", "mini-offline", "mini-workspace", "one-mac", "standings",
        "stories",
    )},
    "two-macs":("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
                 [exchange("linux-1", "ex-1", "gh", "device-code", "github.com"),
                  broken("macbook-pro", "npm", "you", "The token is refused by registry.npmjs.org")]),
    "sign-ins": ("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
                 [signed_out("linux-1", "claude"), exchange("linux-1", "ex-1", "claude", "paste", "claude.com", asking=True)]),
    "calm": ("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
             [dropped("macbook-pro", "jobs", "com.example.brain.x-snapshot")]),
    "machinefile-kept": ("two-macs", "homebrew", applied(["lazygit", "bat"]), []),
    "machinefile-differs": ("two-macs", "homebrew", applied(["lazygit", "bat", "ripgrep", "fd"]), []),
    "ranged-apps": ("two-macs", "homebrew",
                    applied(["lazygit", "{ name: node, version: '24' }", "{ name: go, version: '1.25' }",
                             "{ name: ripgrep, optional: true }"]), []),
    "presence-refused": ("two-macs", None, {}, [switched("presence", "refused")]),
    "stories-file": ("stories", "homebrew", machine("macbook-pro", ["gh", "lazygit", "ghostty"], "settings: { git-shortcuts: {} }\n"), []),
    "leftovers": ("two-macs", None, {}, [leftover("macbook-pro", "old-vm", 9000000000),
                                         holds_file("macbook-pro", STATION_VM, 1000000)]),
    "old-home-running": ("two-macs", None, {}, [holds_file("macbook-pro", STATION_VM, 1000000),
                                                dropped("macbook-pro", "disk", "files", "~/.tart/vms/linux-dev/disk.img"),
                                                holds_file("macbook-pro", "~/.tart/cache/OCIs/ghcr.io/x/disk.img", 4000000000),
                                                running("macbook-pro", "tart")]),
    "leftover-running": ("two-macs", None, {}, [leftover("macbook-pro", "old-vm", 9000000000),
                                                holds_file("macbook-pro", STATION_VM, 1000000),
                                                started("macbook-pro", "old-vm")]),
    "record-unreadable": ("two-macs", None, {}, [leftover("macbook-pro", "old-vm", 9000000000),
                                                 holds_file("macbook-pro", STATION_VM, 1000000),
                                                 unreadable("macbook-pro", "api")]),
    "share-local": ("two-macs", None, {}, [kept_local("macbook-pro", "linux-1", 5173)]),
    "range-full": ("range-full", None, {}, [listens("macbook-pro", "linux-1", 9000, "node")]),
}


def seeded(world):
    base, fixture, files, edits = WORLDS.get(world, plain(world))
    seed = ROOT / "specs" / "usecases" / "worlds" / f"{base}.json"
    if not files and not edits:
        return seed, fixture
    content = json.loads(seed.read_text())
    content["macs"][content["this_mac"]].setdefault("files", {}).update(files)
    for edit in edits:
        edit(content)
    copy = temporary("window-seed-") / seed.name
    copy.write_text(json.dumps(content))
    return copy, fixture


if __name__ == "__main__":
    print(world_binary())
