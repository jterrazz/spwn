"""The world build of station and the sandbox one of its doors runs in, shared by the recorder and the bridge.

Run alone, it prints the path of the world build of this tree, made by `go build -tags world` once per state of
the Go sources and kept as <STATION_BENCH or the temp folder>/worlds/station-world-<hash>, then reused by every
copy of the repository whose sources hash the same. In a Linux VM only, as the door asks the spec build's
environment: HOME, PATH and every station path in a temporary folder, never the user's.

WORLDS is the one list of the worlds the window records, each a world of specs/usecases/worlds/ and what a page
needs that it lacks: machinefiles on This Mac, the fake binaries of a fixture of specs/usecases/__fixtures__/, and
edits of its machines (a sign-in in progress, a program signed out or broken, an accounts switch the fake reads in
apps/engine/world/accounts.go, as the presence check a grant asks, a failed sign-in or a failing background task
dropped, what a Mac's engine keeps of Awake, what a Mac's journal keeps, a VM working hard on every core of its Mac, a workspace whose create failed at its boot); a world of WORLDS may be a world of specs/usecases/worlds/ under another name, seeded otherwise. A world no
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


BOOT_CONSOLE = [
    "[    0.000000] Linux version 6.8.0-49-generic",
    "[    1.842113] EXT4-fs (vda1): mounted filesystem",
    "[    2.371005] cloud-init[498]: running init-local",
    "[    4.102771] Kernel panic - not syncing: VFS: unable to mount root fs",
]


def failed_at_boot(name, vm):
    """A workspace whose create failed at its boot, kept on the record: what `world fault Hypervisor.Boot 1` then
    `machines create --memory 4G` left in world.json over two-macs, its VM's console holding BOOT_CONSOLE beforehand,
    copied from that run in the bench's world sandbox (2026-11-19): the VM's disk and its record with its failed
    birth, the console's last lines the engine kept in it, its clone, and the ports its rights reserved."""
    def edit(content):
        held = content["macs"][name]
        held["vms"][vm] = {
            "console": BOOT_CONSOLE,
            "kept": {
                "image": "base-ubuntu", "version": "2026.11.01", "lifetime": "ephemeral", "network": "nat",
                "env": {
                    "EXPO_PACKAGER_PROXY_URL": "https://macbook-pro.tail1234.ts.net:40000",
                    "STATION_PUBLIC_HOST": "macbook-pro.tail1234.ts.net",
                    "__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS": "macbook-pro.tail1234.ts.net",
                },
                "created_at": "2026-11-04T10:00:19Z",
                "birth": {
                    "progress": 400, "step": "booting", "position": 2, "steps": 4, "at": "2026-11-04T10:00:19Z",
                    "sentence": "Failed at boot 2 of 4", "failed": True,
                    "cause": f"{vm} did not boot: the hypervisor refused to boot it, as the world scripted it",
                    "last_lines": BOOT_CONSOLE[1:],
                },
            },
            "disk": True, "born": 1, "made": True,
            "size": {"CPUs": 4, "Memory": 4294967296, "Disk": 20000000000},
            "guest": {"fresh": True},
        }
        held["cloned"] = {**held.get("cloned", {}), vm: 1}
        held["stores"]["reservations"] += [
            {"vm": vm, "guest_port": 8081, "host_port": 40000, "by": "rights", "http": True,
             "reserved_at": "2026-11-04T10:00:19Z"},
            {"vm": vm, "guest_port": 5173, "host_port": 40003, "by": "rights", "http": True,
             "reserved_at": "2026-11-04T10:00:19Z"},
        ]
    return edit


def installed_for(mac, machinefile, package, method):
    """A package the ledger says station installed for `machinefile`: when the file lists it no longer, it is an extra
    Apply leaves (specs/usecases/machinefile/drift/an-extra-stays-at-a-bare-apply.spec.yaml)."""
    def edit(content):
        content["macs"][mac].setdefault("stores", {}).setdefault("ledger", []).append(
            {"package": package, "method": method, "machinefile": machinefile, "at": "2026-11-03T09:00:00Z"})
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


def kept_awake(name, **kept):
    """What a Mac's engine holds of Awake between two commands (a keep-awake running, the end of the last one), as
    the engine keeps it in the world."""
    def edit(content):
        content["macs"][name]["engine"]["awake"] = kept
    return edit


def journaled(mac, lines):
    """What a Mac's journal keeps, as the engine's own stories seed it (specs/usecases/journal/read/): each line a
    `journal.Line` (apps/engine/core/journal/journal.go) about the Mac or one of its VMs, a gone machine's kept by the
    Mac that held it."""
    def edit(content):
        content["macs"][mac].setdefault("stores", {})["journal"] = lines
    return edit


def engine_logged(mac, lines):
    """The lines a Mac's engine log holds, as the world seeds them (specs/usecases/engine/log/): a text shaped like a
    secret is withheld by the engine when it answers."""
    def edit(content):
        content["macs"][mac]["engine_log"] = lines
    return edit


def kept_line(at, on, text, **fields):
    """A line done by you, as the engine words it ("gate-2 deleted"), about `on`, the Mac or one of its VMs."""
    return {"At": at, "Machine": on, "Text": text, "Detail": "", "By": "you", "Outcome": "done", **fields}


def about(kind, name):
    return {"Kind": kind, "Name": name}


def night_line(night, at, on, package, **fields):
    """A line of the night run started at `night`, about the app `package` on `on`."""
    return kept_line(at, on, package, By="the night", Subject=about("app", package), Night=night, **fields)


def moved(night, at, on, package, versions):
    return night_line(night, at, on, package, Upgrade={"From": versions[0], "To": versions[1]})


def failed(night, at, on, package, args):
    """A night's update that failed on `on`, its Retry the apps.update `args` name, sent at that machine."""
    return night_line(night, at, on, package, Detail="download timed out", Outcome="failed",
                      Retry={"Call": "apps.update", "Args": args, "On": on})


STORIES_NIGHT = "2026-11-07T03:00:00Z"
NODE = ("24.11.0", "24.11.1")
UV = ("0.9.7", "0.9.9")


def stories_night(machines, package, versions, minute):
    return [moved(STORIES_NIGHT, f"2026-11-07T03:{minute}:00Z", on, package, versions) for on in machines]


STORIES_JOURNAL = [
    journaled("macbook-pro", [
        *stories_night(["macbook-pro", "agent-7f", "api", "linux-1"], "node", NODE, 12),
        moved(STORIES_NIGHT, "2026-11-07T03:12:00Z", "macbook-pro", "gh", ("2.97.0", "2.98.0")),
        *stories_night(["agent-7f", "api", "linux-1"], "uv", UV, 13),
        night_line(STORIES_NIGHT, "2026-11-07T03:14:00Z", "macbook-pro", "orbstack", Detail="updates itself",
                   Outcome="held"),
        failed(STORIES_NIGHT, "2026-11-07T03:15:00Z", "linux-1", "ripgrep", {"Names": "ripgrep"}),
    ]),
    journaled("mac-mini", [
        kept_line("2026-11-07T09:12:00Z", "gate-2", "gate-2 deleted", Call="machines.delete", Detail="8 GB freed",
                  Subject=about("machine", "gate-2")),
        kept_line("2026-11-07T09:05:00Z", "infra", "infra restarted", Call="machines.restart",
                  Subject=about("machine", "infra")),
        *stories_night(["mac-mini", "infra"], "node", NODE, 12),
        *stories_night(["infra", "review-bot"], "uv", UV, 13),
        kept_line("2026-11-06T18:02:00Z", "linux-3", "linux-3 created", Call="machines.create-workspace",
                  By="agent-7f", Purpose="review the docs lot", Subject=about("machine", "linux-3")),
    ]),
]

TWO_MACS_NIGHT = "2026-11-04T03:00:00Z"

TWO_MACS_JOURNAL = [
    journaled("macbook-pro", [
        moved(TWO_MACS_NIGHT, "2026-11-04T03:12:00Z", "macbook-pro", "node", NODE),
        moved(TWO_MACS_NIGHT, "2026-11-04T03:12:00Z", "linux-1", "node", NODE),
        failed(TWO_MACS_NIGHT, "2026-11-04T03:15:00Z", "macbook-pro", "mise", {"Managers": "mise"}),
    ]),
]


def released(name, installed, tag, kept):
    """station's release `tag` on GitHub with every binary attached, a Mac's engine at version `installed`, and what
    that engine kept of its last asks (when GitHub last answered, the refusal since), so its job asks nothing at the
    first turn."""
    url = f"https://github.com/jterrazz/spwn/releases/tag/{tag}"
    assets = ["station-linux-amd64", "station-linux-arm64", "station-darwin-amd64", "station-darwin-arm64"]

    def edit(content):
        content["releases"] = [{"tag": tag, "url": url, "assets": assets}]
        content["macs"][name]["engine"]["version"] = installed
        content["macs"][name].setdefault("stores", {})["release"] = {
            "Read": kept["read"], "Latest": {"Tag": tag, "URL": url}, "Refused": kept.get("refused")}
    return edit


def never_asked(name, installed, tag):
    """station's release `tag` on GitHub and a Mac's engine at version `installed`, stopped before its first ask: it
    kept nothing."""
    def edit(content):
        released(name, installed, tag, {"read": ""})(content)
        del content["macs"][name]["stores"]["release"]
        content["macs"][name]["engine"].update({"stopped": True, "stopped_at": "2026-11-04T09:40:00Z", "stopped_by": "the menu bar"})
    return edit


def first_refused(name, installed, tag):
    """station's release `tag` on GitHub, a Mac's engine at version `installed` whose first ask GitHub refuses."""
    def edit(content):
        released(name, installed, tag, {"read": ""})(content)
        del content["macs"][name]["stores"]["release"]
        content["script"] = [{"call": "Releases.Latest", "times": 1}]
    return edit


def working_hard(mac, vm, cpu, busy, top):
    """A VM given every core of its Mac and busy on all of them: its guest's meter at `cpu` % of its CPUs, its
    Virtualization process the same share of its Mac, the Mac at `busy` % in all, the history its Mac's engine kept of
    both around those levels, and the VM's busiest processes by CPU and by memory, `top`'s (name, pid, CPU %, resident
    bytes) in that order."""
    def edit(content):
        held = content["macs"][mac]
        cores = held["self"]["hardware"]["CPUs"]
        record = held["vms"][vm]
        record["size"]["CPUs"] = cores
        held["meter"]["processes"][str(record["pid"])]["cpu"] = cpu
        held["meter"]["per_second"]["cpu_busy"] = round(busy * 10)
        held["engine"]["past"][vm]["base"], held["engine"]["past"][mac]["base"] = cpu, busy
        meter = content["macs"][vm]["meter"]
        meter["sample"]["cpus"] = cores
        meter["per_second"]["cpu_busy"] = round(cpu * 10)
        meter["top"] = {
            "cpu": [{"pid": pid, "name": name, "value": share} for name, pid, share, _ in top],
            "memory": [{"pid": pid, "name": name, "value": size}
                       for name, pid, _, size in sorted(top, key=lambda one: -one[3])],
        }
    return edit


BUSIEST = [("node", 1190, 38.5, 2147483648), ("rustc", 2214, 21.2, 1503238553), ("cc1plus", 2231, 14.7, 943718400),
           ("esbuild", 2302, 9.4, 314572800), ("git", 2340, 4.1, 104857600)]


def plain(name):
    return (name, None, {}, [])


STATION_VM = "~/.local/share/station/vms/vms/linux-1/disk.img"


WORLDS = {
    **{name: plain(name) for name in (
        "bare-mac", "disk-low", "engine-stopped", "mini-offline", "mini-workspace", "one-mac", "standings",
    )},
    "creating": ("creating", None, {}, []),
    "stories": ("stories", None, {}, STORIES_JOURNAL),
    "two-macs":("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
                 [exchange("linux-1", "ex-1", "gh", "device-code", "github.com"),
                  broken("macbook-pro", "npm", "you", "The token is refused by registry.npmjs.org"),
                  *TWO_MACS_JOURNAL,
                  engine_logged("macbook-pro", ["engine started", "nightly run started", "ping https://u:hunter2@example.com/x ok",
                                                "nightly run done"]),
                  engine_logged("mac-mini", ["engine started", "waiting for the next night run"])]),
    "sign-ins": ("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
                 [signed_out("linux-1", "claude"), exchange("linux-1", "ex-1", "claude", "paste", "claude.com", asking=True)]),
    "calm": ("two-macs", "homebrew", machine("macbook-pro", ["lazygit", "bat"]),
             [dropped("macbook-pro", "jobs", "com.example.brain.x-snapshot")]),
    "machinefile-kept": ("two-macs", "homebrew", applied(["lazygit", "bat"]), []),
    "machinefile-extra": ("two-macs", "homebrew", applied(["lazygit"]),
                          [installed_for("macbook-pro", "macbook-pro", "bat", "brew")]),
    "machinefile-differs": ("two-macs", "homebrew", applied(["lazygit", "bat", "ripgrep", "fd"]), []),
    "ranged-apps": ("two-macs", "homebrew",
                    applied(["lazygit", "{ name: node, version: '24' }", "{ name: go, version: '1.25' }",
                             "{ name: ripgrep, optional: true }"]), []),
    "presence-refused": ("two-macs", None, {}, [switched("presence", "refused")]),
    "working-hard": ("stories", None, {}, [working_hard("macbook-pro", "linux-1", 92, 95, BUSIEST)]),
    "stories-file":("stories", "homebrew", machine("macbook-pro", ["gh", "lazygit", "ghostty"], "settings: { git-shortcuts: {} }\n"), []),
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
    "create-failed": ("two-macs", None, {}, [failed_at_boot("macbook-pro", "linux-2")]),
    "range-full": ("range-full", None, {}, [listens("macbook-pro", "linux-1", 9000, "node")]),
    "kept-four-hours": ("two-macs", None, {}, [kept_awake("macbook-pro", session={
        "lid": False, "battery": False, "started_at": "2026-11-04T10:00:00Z", "until": "2026-11-04T14:00:00Z",
        "by": "station awake 4h", "for": 4 * 3600 * 10**9})]),
    "awake-stopped": ("two-macs", None, {}, [kept_awake("macbook-pro", last_end={
        "at": "2026-11-04T09:58:00Z", "reason": "battery", "detail": "battery 18 %"})]),
    "release-behind": ("two-macs", None, {}, [released("macbook-pro", "v0.21.0", "v0.22.0",
                                                       {"read": "2026-11-04T09:00:00Z"})]),
    "release-unreachable": ("two-macs", None, {}, [released("macbook-pro", "v0.21.0", "v0.22.0", {
        "read": "2026-11-04T02:00:00Z",
        "refused": {"At": "2026-11-04T09:50:00Z", "Message": "GitHub: dial tcp: connection refused"}})]),
    "release-never-asked": ("two-macs", None, {}, [never_asked("macbook-pro", "v0.21.0", "v0.22.0")]),
    "release-first-refused": ("two-macs", None, {}, [first_refused("macbook-pro", "v0.21.0", "v0.22.0")]),
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
