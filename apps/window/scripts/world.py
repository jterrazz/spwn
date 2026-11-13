"""The world build of station and the sandbox one of its doors runs in, shared by the recorder and the bridge.

Run alone, it prints the path of the world build of this tree, made by `go build -tags world` once per state of
the Go sources and kept as <STATION_BENCH or the temp folder>/worlds/station-world-<hash>, then reused by every
copy of the repository whose sources hash the same. In a Linux VM only, as the door asks the spec build's
environment: HOME, PATH and every station path in a temporary folder, never the user's.

SEEDS adds to a world what the specs' worlds lack for a page: machinefiles on This Mac, and the fake binaries
of a fixture of specs/usecases/__fixtures__/; a world of SEEDS may be a world of specs/usecases/worlds/ under
another name, seeded otherwise.

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


def world_binary():
    digest = hashlib.sha256()
    modules = [ROOT / "go.work", *sorted(ROOT.glob("go.work.sum"))]
    for module in ("apps/engine", "catalog"):
        modules += [ROOT / module / "go.mod", *sorted((ROOT / module).glob("go.sum"))]
    sources = [*modules, *sorted((ROOT / "apps" / "engine").rglob("*.go")), *sorted((ROOT / "catalog").rglob("*"))]
    sources = [source for source in sources if source.is_file()]
    for source in sources:
        digest.update(str(source.relative_to(ROOT)).encode())
        digest.update(source.read_bytes())
    kept = pathlib.Path(os.environ.get("STATION_BENCH") or tempfile.gettempdir()) / "worlds"
    kept.mkdir(parents=True, exist_ok=True)
    binary = kept / f"station-world-{digest.hexdigest()[:16]}"
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
name: jterrazz-base
description: What every machine of the owner carries
apps: [jq]
"""


def machine(name, apps, settings=""):
    return {
        "machinefiles/jterrazz-base.machine.yaml": BASE,
        f"machinefiles/{name}.machine.yaml": "machinefile: station.machinefile/v1\n"
        f"name: {name}\ndescription: The owner's Mac\nmachine: {{ kind: computer, system: {{ os: macos }} }}\n"
        f"extends: [jterrazz-base]\napps: [{', '.join(apps)}]\n" + settings,
    }


LOCK = """lock: station.lock/v1
machinefile: jterrazz-macbook-pro
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
    return {**machine("jterrazz-macbook-pro", apps), "machinefiles/jterrazz-macbook-pro.lock": LOCK}


SEEDS = {
    "two-macs": ("two-macs", "homebrew", machine("jterrazz-macbook-pro", ["lazygit", "bat"])),
    "machinefile-kept": ("two-macs", "homebrew", applied(["lazygit", "bat"])),
    "machinefile-differs": ("two-macs", "homebrew", applied(["lazygit", "bat", "ripgrep", "fd"])),
    "stories-file": ("stories", "homebrew", machine("macbook-pro", ["gh", "lazygit", "ghostty"], "settings: { git-shortcuts: {} }\n")),
}


def seeded(world):
    base, fixture, files = SEEDS.get(world, (world, None, {}))
    seed = ROOT / "specs" / "usecases" / "worlds" / f"{base}.json"
    if not files:
        return seed, fixture
    content = json.loads(seed.read_text())
    content["macs"][content["this_mac"]].setdefault("files", {}).update(files)
    copy = temporary("window-seed-") / seed.name
    copy.write_text(json.dumps(content))
    return copy, fixture


if __name__ == "__main__":
    print(world_binary())
