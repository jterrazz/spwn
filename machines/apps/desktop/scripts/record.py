"""Record the engine's real answers the window's pages read, over the worlds of specs/usecases/worlds/.

Usage: record.py [--call=<call>...] [<world>...], every world of world.py's WORLDS when none is named, in a Linux VM
only: it asks `station guest serve --stdio` of the world build (world.py), the door
specs/architecture/answers_test.go asks, over a fresh copy of each world in a temporary HOME (one per request,
so no answer depends on what another request did to the world), and writes one file per call:
apps/desktop/src/fixtures/worlds/<world>/<call>.json, every request of that call the pages make in that world,
with its answers. The recorder's tables are the one list of what is asked: a page that reads a new call, or the same
call with a new input, adds its requests to `asks` (`follows`, when they depend on another answer; ALSO, when one
world's spec alone makes it) and records again the worlds its specs use.

A gesture of GESTURES is recorded in a fresh world of its own: the gesture, then every read of that world asked
again in the same door, one after the other, with the reads of a machine the gesture created, and written to
<world>/gestures/<call>.json with the reads whose answer it changed; the replay transport plays it, then answers
those reads as they were after it. The reads a gesture names `during` are asked in that door once its first frame is
in, while it runs, and written under `during` as they answered; the replay answers them so while it plays the
gesture (scripts/door.py, `ask`).

One path records every world, one world or one call: a world is always asked whole, and `--call=<call>` writes only
that call's files, <world>/<call>.json and <world>/gestures/<call>.json, in the worlds named or every one. A world
WORLDS adds is recorded by the same command, its folder made; a run over every world removes the folder of a world
WORLDS no longer names.

`--if-moved`, the form `make generate` runs, records every world unless nothing the run reads and nothing it writes
moved since one that succeeded: `key` hashes them by content, and a run that succeeds leaves its key in
.artifacts/window-record.stamp, outside the tracked tree, so a fresh clone always records. A hit prints one line and its
key; WINDOW_RECORD_FORCE=1 records all the same.

Two runs over the same tree write the same bytes. The world build's clock moves when a watch waits on it
(apps/engine/wire/world_on.go), and what a watch streams before it settles comes in the scheduler's order, so the
recorder keeps only what no run decides: of a watch its first answer and its last, and of a watch of SETTLED (one
whose first answer races its walk, disk.read) its last alone; of overview.read, its last alone too unless it streams a
birth: its summary answers once per part (each section, the machines, the other Macs, the readiness) in the order the
parts arrive, and the engine ends the watch once every part is in (apps/engine/overview/calls/compose.go), so the
answer before its done is the whole Overview, which the recorder waits for, never letting go of it on silence; a
gesture's reads in the world's order of asks, judged changed by `compared`, never by the clock; no read of TIMED after
a gesture, but the activity of the VM a gesture of MEASURED stopped, which measures nothing; and, as the file's last recording holds them, the NOISE field of a gesture (the process number the host
gave) and the CLOCKED field of an answer (the world's clock: settings.matrix's Read, awake.read's Now).

In a world run that door is a Mac's engine socket: a frame for another Mac, or for a VM of another Mac, rides the
world's far dialer and is answered there, so every Mac machines.list lists and every VM of each is asked, by name.

A sign-in call names its VM in `Machine` to this Mac's engine and takes no target: accounts.list is asked for This Mac
and once per VM of every Mac, accounts.login and accounts.logout carry their `Machine`, and the `sign-ins` world
holds the login whose code is pasted back. The vault is This Mac's alone: vault.list and vault.new are asked with no target, and
grants.new once per entry it lists. A gesture's input is written with its recording, so a pasted code or a typed
password is a plain word (`good`, `nope`), never one shaped like a token.
"""

import concurrent.futures
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys

import door as doors
from door import UNENDED, Door, born, callOf
from world import ROOT, WORLDS, build_digest, seeded, world_binary

out_root = ROOT / "apps" / "desktop" / "src" / "fixtures" / "worlds"
GRAPHS = (360 * 10**9, 3600 * 10**9, 86400 * 10**9)
SETTLED = {"apps.list", "settings.list", "disk.read"}
TIMED = {"activity.read", "engine.watch"}
MEASURED = {"machines.stop"}
NOISE = {"engine.start": "PID"}
CLOCKED = {"settings.matrix": "Read", "awake.read": "Now"}
MACHINES = {"watch": "machines.list", "in": {}}
STAMP = ROOT / ".artifacts" / "window-record.stamp"
FORCE = "WINDOW_RECORD_FORCE"


def serve(seed, requests, fixture=None):
    door = Door(seed, fixture)
    answered = door.ask(requests)
    door.close()
    return door.station, answered


def last_out(frames):
    outs = [frame["out"] for frame in frames if "out" in frame]
    return outs[-1] if outs else None


def settled(frames):
    last = [frame for frame in frames if "out" in frame][-1:]
    return [frame for frame in frames if "out" not in frame or frame in last]


def at(name):
    return {"target": {"name": name}}


def macsOf(listed):
    out = last_out(listed) or {}
    return [mac["Name"] for mac in (out.get("All") or {}).get("Macs") or [] if mac.get("Name")]


def vmsOf(listed):
    """Every VM of every Mac: their sign-ins are asked of this Mac's engine, which says what a far Mac's VM allows."""
    out = last_out(listed) or {}
    return [vm["Name"] for mac in (out.get("All") or {}).get("Macs") or [] for vm in mac.get("VMs") or []]


def asks(listed):
    """The reads the window asks of every Mac machines.list lists and of every VM of each. This Mac's engine is the
    socket's own, so a read of it names no target, and a far Mac or a VM is named in the target, as targetOf
    (src/domain/machines/machines.ts) sends it; a read two pages ask is asked once. A drift names its machine in its
    input and no target, as driftAsked sends it. A VM's activity and its log are asked at its Mac, as useActivity
    and VmLog send them. A Mac's Awake sheet watches its awake.read, of This Mac's engine for This Mac and named for a
    far one. The New machine sheet watches machines.new of the socket's engine, opened anywhere or on one
    Mac machines.list lists, as askedOf sends it; a sheet opened from a machine file names it in From, a request of
    ALSO."""
    here = thisMac(listed)
    macs = allMacs(listed)
    far = [mac["Name"] for mac in macs if not mac.get("ThisMac")]
    vms = [(mac, vm) for mac in macs for vm in mac.get("VMs") or []]
    targets = [onMac(mac) for mac in macs] + [at(vm["Name"]) for _, vm in vms]
    return list({keyOf(request): request for request in [
        {"watch": "overview.read", "in": {}},
        {"watch": "overview.read", "in": {"Elsewhere": True}},
        {"call": "engine.read", "in": {}},
        {"call": "engine.read-release", "in": {}},
        {"call": "ports.list", "in": {"AllMachines": True}},
        *[{"call": "ports.list", "in": {"Of": name}} for name in [*macsOf(listed), *vmsOf(listed)]],
        {"call": "images.list", "in": {}},
        {"watch": "apps.list", "in": {}},
        {"watch": "settings.list", "in": {}},
        {"call": "settings.matrix", "in": {}},
        {"call": "apps.matrix", "in": {}},
        {"call": "machinefile.explain", "in": {}},
        {"call": "machinefile.files", "in": {}},
        {"watch": "engine.watch", "in": {"machines": []}},
        *[{"watch": "engine.watch", "in": {"machines": [name]}} for name in [*far, *[vm["Name"] for _, vm in vms]]],
        *[{kind: call, "in": {}, **target} for target in targets for kind, call in (
            ("watch", "overview.read"), ("watch", "apps.list"), ("watch", "settings.list"),
            ("call", "machinefile.plan"), ("call", "machinefile.explain"), ("watch", "disk.read"),
        )],
        {"call": "machinefile.drift", "in": {}},
        *[{"call": "machinefile.drift", "in": {"Machine": name}} for name in [*far, *[vm["Name"] for _, vm in vms]]],
        *[{"call": call, "in": {"Machinefile": "base-macos"}}
          for _ in here for call in ("machinefile.drift", "machinefile.plan")],
        *[{"watch": "activity.read", "in": {"graph": graph}, **target} for target in targets for graph in GRAPHS],
        *[{"watch": "activity.read", "in": {"graph": graph, "of": vm["Name"], "machine": {"Name": vm["Name"], "On": mac["Name"]}}, **onMac(mac)}
          for mac, vm in vms for graph in GRAPHS],
        *[{"watch": "activity.read", "in": {"history": True}, **target} for target in targets],
        *[{"watch": "activity.read", "in": {"history": True, "of": vm["Name"], "machine": {"Name": vm["Name"], "On": mac["Name"]}}, **onMac(mac)}
          for mac, vm in vms],
        *[{"call": "machines.log", "in": {"VM": vm["Name"]}, **onMac(mac)} for mac, vm in vms],
        *[{"call": "machines.ssh", "in": {"Name": name}} for name in vmsOf(listed)],
        *[{"watch": "accounts.list", "in": {}} for _ in here],
        *[{"watch": "accounts.list", "in": {"Machine": name}} for name in vmsOf(listed)],
        *[{"watch": "vault.list", "in": {}} for _ in here],
        *[{"watch": "vault.new", "in": {}} for _ in here],
        *[{"watch": "awake.read", "in": {}, **onMac(mac)} for mac in macs],
        {"watch": "machines.new", "in": {}},
        *[{"watch": "machines.new", "in": {"On": mac}} for mac in macsOf(listed)],
    ]}.values())


ALSO = {
    "stories": [
        {"call": "machinefile.explain", "in": {"Machinefile": "linux-dev"}},
        {"watch": "machines.new", "in": {"From": "linux-dev"}},
        {"watch": "machines.new", "in": {"From": "base-ubuntu"}},
    ],
}


def asked_in(world, listed):
    """Every read the window asks in a world: those `asks` draws from its machines, then the ones of ALSO, a read a
    spec of that world makes that no listing gives (a sheet opened from one machine file)."""
    return list({keyOf(request): request for request in [*asks(listed), *ALSO.get(world, [])]}.values())


def follows(requests, answered):
    """The reads asked of an answer: a drift per machine file machinefile.files names, and the grant picker's offer
    per entry of this Mac's vault."""
    listed = [last_out(frames) for request, frames in zip(requests, answered)
              if request.get("call") == "machinefile.files" and "target" not in request]
    names = [file["Name"] for out in listed if out for file in out.get("Files") or [] if file.get("Name")]
    vaults = [last_out(frames) for request, frames in zip(requests, answered) if request.get("watch") == "vault.list"]
    entries = [entry["ID"] for out in vaults if out for entry in out.get("Entries") or [] if entry.get("ID")]
    asked_already = {keyOf(request) for request in requests}
    more = [
        *[{"call": "machinefile.drift", "in": {"Machinefile": name}} for name in dict.fromkeys(names)],
        *[{"watch": "grants.new", "in": {"Entry": entry}} for entry in dict.fromkeys(entries)],
    ]
    return [request for request in more if keyOf(request) not in asked_already]


def asked(seed, requests, fixture):
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        return list(pool.map(lambda request: serve(seed, [request], fixture)[1][0], requests))


STOP_LINUX_1 = {"call": "machines.stop", "in": {"VM": "linux-1", "Confirm": "yes"}}

GESTURES = {
    "two-macs": [
        STOP_LINUX_1,
        {"call": "machines.start", "in": {"VM": "linux-1"}, "following": STOP_LINUX_1},
        {"call": "machines.start", "in": {"VM": "api"}},
        {"call": "machines.delete", "in": {"VM": "api", "Confirm": "api"}},
        {"call": "machines.restart", "in": {"Name": "infra", "Confirm": "infra"}, **at("mac-mini")},
        {"call": "apps.update", "in": {"Managers": ["mise"]}},
        {"call": "awake.keep", "in": {"For": 14400000000000}},
        {"call": "disk.free", "in": {}},
        {"call": "disk.free", "in": {"Apps": ["go"]}},
        {"call": "disk.free", "in": {}, **at("linux-1")},
        {"call": "disk.move", "in": {"Folder": "~/.tart"}},
        {"call": "disk.erase", "in": {"App": "ollama", "Confirm": "ollama"}},
        {"call": "disk.erase", "in": {"App": "orbstack", "Confirm": "orbstack"}},
        {"call": "images.delete", "in": {"Image": "base-ubuntu", "Version": "2026.10.20", "Confirm": "yes"}},
        {"call": "ports.stop-sharing", "in": {"VM": "linux-1", "Port": "5173", "Confirm": "yes"}},
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "5173", "To": "local"}},
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "5173", "To": "public"}},
        {"watch": "accounts.login", "in": {"Program": "gh", "Machine": "linux-1"}},
        {"watch": "accounts.login", "in": {"Program": "npm"}},
        {"call": "accounts.open", "in": {"Machine": "linux-1", "Exchange": "ex-1"}},
        {"call": "accounts.logout", "in": {"Program": "claude", "Machine": "linux-1", "Confirm": "yes"}},
        {"call": "background.remove", "in": {"Label": "com.example.brain.x-snapshot", "Confirm": "yes"}},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "base-ubuntu", "Lifetime": "ephemeral"}, "Confirm": "yes"}, **at("mac-mini")},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "base-ubuntu", "Lifetime": "ephemeral", "Size": {"MemoryBytes": 4294967296}}, "Confirm": "yes"}},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "infra", "From": "base-ubuntu", "Lifetime": "ephemeral"}, "Confirm": "yes"}, **at("mac-mini")},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "linux-2", "From": "base-ubuntu", "Lifetime": "ephemeral", "Purpose": "x" * 121}, "Confirm": "yes"}, **at("mac-mini")},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "linux-2", "From": "base-ubuntu", "Lifetime": "ephemeral", "Purpose": "Lot shell"}, "Confirm": "yes"}, **at("mac-mini"),
         "during": [MACHINES, {"watch": "overview.read", "in": {}, **at("linux-2")}]},
        {"call": "machines.create-server", "in": {"Name": "web", "Image": "base-ubuntu", "Size": {"MemoryBytes": 4294967296}, "Confirm": "yes"}},
        {"call": "machines.add", "in": {"Name": "mac-mini"}},
        {"call": "vault.remove", "in": {"ID": "pw-2", "Confirm": "yes"}},
        {"call": "grants.remove", "in": {"Entry": "pw-2", "To": "linux-1", "Confirm": "yes"}},
        {"call": "grants.add", "in": {"Entry": "pw-1", "To": "linux-1"}},
        {"call": "grants.add", "in": {"Entry": "pw-2", "To": "linux-1"}},
        {"call": "vault.add", "in": {"Label": "GitHub sign-in", "Program": "gh", "From": "sign-in"}},
        {"call": "vault.add", "in": {"Label": "Cloudflare deploy", "Program": "wrangler", "Kind": "token", "From": "typed", "Value": "good"}},
        {"call": "vault.add", "in": {"Label": "", "Program": "npm", "Kind": "token", "From": "typed", "Value": "good"}},
        {"call": "vault.add", "in": {"Label": "GitHub deploy", "Program": "npm", "Kind": "token", "From": "typed", "Value": "good"}},
        {"call": "vault.add", "in": {"Label": "npm deploy", "Program": "npm", "Kind": "token", "From": "typed", "Value": ""}},
    ],
    "range-full": [
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "9000"}},
    ],
    "share-local": [
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "5173", "To": "private"}},
    ],
    "presence-refused": [
        {"call": "grants.add", "in": {"Entry": "pw-1", "To": "linux-1"}},
    ],
    "stories": [
        {"call": "awake.off", "in": {}},
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "4321"}},
        {"call": "ports.share", "in": {"VM": "linux-3", "Port": "5432"}, **at("mac-mini")},
        {"call": "ports.declare", "in": {"Machine": "macbook-pro", "Port": "5000", "Via": "lan"}},
        {"call": "machines.restart", "in": {"Name": "infra", "Confirm": "infra"}, **at("mac-mini")},
    ],
    "sign-ins": [
        {"watch": "accounts.login", "in": {"Program": "claude", "Machine": "linux-1"}},
        {"call": "accounts.open", "in": {"Machine": "linux-1", "Exchange": "ex-1"}},
        {"call": "accounts.answer", "in": {"Machine": "linux-1", "Exchange": "ex-1", "Code": "good"}},
        {"call": "accounts.answer", "in": {"Machine": "linux-1", "Exchange": "ex-1", "Code": "nope"}},
    ],
    "engine-stopped": [
        {"call": "machines.start", "in": {"VM": "linux-1"}},
        {"call": "engine.start", "in": {}},
    ],
    "one-mac": [
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "base-ubuntu", "Lifetime": "ephemeral"}, "Confirm": "yes"}},
        {"call": "machines.add", "in": {"Name": "mac-mini"}},
    ],
    "creating": [
        {"call": "machines.delete", "in": {"VM": "linux-5", "Confirm": "linux-5"}},
    ],
    "machinefile-differs": [
        {"call": "machinefile.apply", "in": {"Confirm": "yes"}},
        {"call": "machinefile.apply", "in": {"Confirm": "yes"}, **at("api")},
        {"call": "background.remove", "in": {"Label": "com.example.brain.x-snapshot", "Confirm": "yes"}},
        {"call": "machinefile.upgrade", "in": {}},
        {"call": "machinefile.upgrade", "in": {}, **at("api")},
    ],
    "machinefile-kept": [
        {"call": "machinefile.apply", "in": {"Confirm": "yes"}},
        {"call": "machinefile.upgrade", "in": {}},
    ],
    "disk-low": [
        {"call": "disk.free", "in": {}},
        {"call": "disk.move", "in": {"Folder": "~/.tart"}},
    ],
    "leftovers": [
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "old-vm"]}},
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "old-vm"], "Confirm": "base-ubuntu@2026.10.20, old-vm"}},
    ],
    "old-home-running": [
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "~/.tart"]}},
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "~/.tart"], "Confirm": "base-ubuntu@2026.10.20, ~/.tart"}},
    ],
    "leftover-running": [
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "old-vm"]}},
    ],
    "record-unreadable": [
        {"call": "disk.free-unused", "in": {"Names": ["base-ubuntu@2026.10.20", "old-vm"]}},
    ],
}


def bounded(frames):
    outs = [index for index, frame in enumerate(frames) if "out" in frame]
    ends = {outs[0], outs[-1]} if outs else set()
    return [frame for index, frame in enumerate(frames) if "out" not in frame or index in ends]


def kept(request, frames):
    if "watch" not in request:
        return frames
    whole = callOf(request) in UNENDED and not born(frames)
    return settled(frames) if callOf(request) in SETTLED or whole else bounded(frames)


def keyOf(request):
    return json.dumps(request, sort_keys=True)


def allMacs(listed):
    out = last_out(listed) or {}
    return [mac for mac in (out.get("All") or {}).get("Macs") or [] if mac.get("Name")]


def thisMac(listed):
    return [mac for mac in allMacs(listed) if mac.get("ThisMac")]


def onMac(mac):
    """The target a read of a Mac names, as targetOf sends it: none for This Mac, the Mac's name for a far one."""
    return {} if mac.get("ThisMac") else at(mac["Name"])


def without(value, field):
    if isinstance(value, dict):
        return {key: without(item, field) for key, item in value.items() if key != field}
    if isinstance(value, list):
        return [without(item, field) for item in value]
    return value


def unclocked(request, frames):
    """The frames of a call whose answer carries the world's clock, that field left out wherever it sits."""
    field = CLOCKED.get(callOf(request))
    return without(frames, field) if field else frames


def carry(new, old, field):
    if isinstance(new, dict) and isinstance(old, dict):
        for key in new:
            if key == field and key in old:
                new[key] = old[key]
            elif key in old:
                carry(new[key], old[key], field)
    elif isinstance(new, list) and isinstance(old, list):
        for one, was in zip(new, old):
            carry(one, was, field)


def steadied(was, key, items):
    """An answer of CLOCKED keeps the clock field its last recording `was` holds, so a recording again moves no file
    for time that passed."""
    held = {keyOf(item["request"]): item for item in was.get(key, [])}
    for item in items:
        before = held.get(keyOf(item["request"]))
        if not before:
            continue
        field = CLOCKED.get(callOf(item["request"]))
        if field:
            carry(item["answers"], before["answers"], field)
        for part in ("during", "after"):
            reads = {keyOf(read["request"]): read for read in before.get(part, [])}
            for read in item.get(part, []):
                field, one = CLOCKED.get(callOf(read["request"])), reads.get(keyOf(read["request"]))
                if field and one:
                    carry(read["answers"], one["answers"], field)


def write(path, world, station, key, items, was):
    """One recording file, its clock and NOISE fields kept as `was`, its last recording, holds them."""
    steadied(was, key, items)
    if key == "gestures":
        for item in items:
            steady(item, was)
    head = json.dumps({"world": world, "station": station})[:-1]
    lines = ",\n".join(json.dumps(item, separators=(",", ":")) for item in items)
    rest = '"exchanges": [], "gestures"' if key == "gestures" else '"exchanges"'
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(f'{head}, {rest}: [\n{lines}\n]}}\n')


def compared(request, frames):
    """A read as a gesture's change is judged: a watch by its settled answer, the CLOCKED field left out, so the
    world's clock moving under a gesture changes no read."""
    return unclocked(request, settled(frames) if "watch" in request else frames)


def measured(request, read):
    """Whether a read of TIMED is asked after a gesture all the same: the activity of the VM a gesture of MEASURED
    stopped, which measures nothing, so its answer moves with no clock."""
    return (callOf(request) in MEASURED and callOf(read) == "activity.read"
            and read["in"].get("of") == request["in"].get("VM"))


def based(door, world, reads, before):
    """What every read answers in `door` once the gesture it follows is made: the world a gesture `following` another
    is judged against, as the window holds it once that one has played."""
    [listed] = door.ask([MACHINES])
    answered = dict(before)
    for read in [*reads, *[one for one in asked_in(world, listed) if keyOf(one) not in before]]:
        if callOf(read) not in TIMED:
            [frames] = door.ask([read])
            answered[keyOf(read)] = kept(read, frames)
    return answered


def gesture(world, seed, fixture, request, reads, before):
    """The gesture in a fresh world, then every read in that same world, kept when its answer changed: a watch by
    its settled answer, the one a window shows once it has read again. A read the world did not ask before, of a
    machine the gesture made, is kept whatever it answers. A gesture `following` another is made once that one is,
    in the same world, and judged against what the reads answer then. The reads it names `during` are asked once its
    first frame is in, while it still runs, and kept as it answers them, changed or not."""
    door = Door(seed, fixture)
    following, during = request.get("following"), request.get("during", [])
    request = {key: value for key, value in request.items() if key not in ("following", "during")}
    if following:
        door.ask([following])
        before = based(door, world, reads, before)
    [answers, *heard] = door.ask([request], lambda count: during if count == 1 else [])
    running = [{"request": read, "answers": kept(read, frames)} for read, frames in zip(during, heard)]
    [listed] = door.ask([MACHINES])
    born = [read for read in asked_in(world, listed) if keyOf(read) not in before]
    changed = []
    for read in [*reads, *born]:
        if callOf(read) in TIMED and not measured(request, read):
            continue
        [frames] = door.ask([read])
        frames, was = kept(read, frames), before.get(keyOf(read))
        if was is None or compared(read, frames) != compared(read, was):
            changed.append({"request": read, "answers": frames})
    door.close()
    return {"request": request, "answers": answers, **({"during": running} if running else {}), "after": changed}


def steady(item, was):
    """A gesture's answer keeps the NOISE field its last recording `was` holds: the number the host gave a process."""
    field = NOISE.get(callOf(item["request"]))
    if not field:
        return
    held = [frame["out"][field] for one in was.get("gestures", []) if one["request"] == item["request"]
            for frame in one["answers"] if field in frame.get("out", {})]
    for frame in item["answers"]:
        if held and field in frame.get("out", {}):
            frame["out"][field] = held[0]


def files_of(folder):
    """The recording a world's folder holds, by its path inside it."""
    if not folder.is_dir():
        return {}
    return {path.relative_to(folder).as_posix(): json.loads(path.read_text()) for path in sorted(folder.rglob("*.json"))}


def call_of_file(name):
    return name.removeprefix("gestures/").removesuffix(".json")


def record(world, calls):
    """A world asked whole, every read then every gesture, and its files written: all of them, or those of `calls`."""
    seed, fixture = seeded(world)
    folder = out_root / world
    station, [listed] = serve(seed, [MACHINES], fixture)
    requests = asked_in(world, listed)
    answered = asked(seed, requests, fixture)
    more = follows(requests, answered)
    if more:
        requests, answered = requests + more, answered + asked(seed, more, fixture)
    requests, answered = [MACHINES, *requests], [listed, *answered]
    before = {keyOf(request): kept(request, frames) for request, frames in zip(requests, answered)}
    chosen = [request for request in GESTURES.get(world, []) if not calls or callOf(request) in calls]
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        made = list(pool.map(lambda request: gesture(world, seed, fixture, request, requests, before), chosen))
    files = {}
    for request in requests:
        files.setdefault(f"{callOf(request)}.json", ("exchanges", []))[1].append(
            {"request": request, "answers": before[keyOf(request)]})
    for item in made:
        files.setdefault(f"gestures/{callOf(item['request'])}.json", ("gestures", []))[1].append(item)
    files = {name: file for name, file in files.items() if not calls or call_of_file(name) in calls}
    prior = files_of(folder)
    for name in prior:
        if name not in files and (not calls or call_of_file(name) in calls):
            (folder / name).unlink()
    for name, (key, items) in files.items():
        write(folder / name, world, station, key, items, prior.get(name, {}))
    if (folder / "gestures").is_dir() and not any((folder / "gestures").iterdir()):
        (folder / "gestures").rmdir()
    return f"{world}: {len(requests)} requests, {len(made)} gestures, {len(files)} files written -> {folder}"


def hashed(digest, paths):
    for path in paths:
        digest.update(path.relative_to(ROOT).as_posix().encode() + b"\0")
        digest.update(hashlib.sha256(path.read_bytes()).digest())


def key():
    """Everything a run over every world reads and writes, by content in sorted order and never by date: the world
    build's sources (world.py's build_digest), the Go toolchain that would build it, the scripts of this folder (the
    recorder, world.py's WORLDS and seeds), every world of specs/usecases/worlds/ a seed is read from, the fake
    binaries of every fixture WORLDS names, and every file the recordings' folder holds now."""
    digest = hashlib.sha256(build_digest().encode())
    go = shutil.which("go")
    version = subprocess.run([go, "env", "GOVERSION"], capture_output=True, text=True).stdout if go else "none"
    digest.update(version.encode())
    fixtures = sorted({fixture for _, fixture, _, _ in WORLDS.values() if fixture})
    hashed(digest, [
        *sorted(pathlib.Path(__file__).parent.glob("*.py")),
        *sorted((ROOT / "specs" / "usecases" / "worlds").glob("*.json")),
        *[path for fixture in fixtures
          for path in sorted((ROOT / "specs" / "usecases" / "__fixtures__" / fixture / "bin").rglob("*")) if path.is_file()],
    ])
    digest.update(b"outputs\0")
    hashed(digest, sorted(path for path in out_root.rglob("*") if path.is_file()) if out_root.is_dir() else [])
    return digest.hexdigest()[:16]


USAGE = "usage: record.py [--if-moved] [--call=<call>...] [<world>...], a world of world.py's WORLDS"

if __name__ == "__main__":
    arguments = sys.argv[1:]
    calls = {argument.removeprefix("--call=") for argument in arguments if argument.startswith("--call=")}
    named = [argument for argument in arguments if not argument.startswith("--")]
    moved = "--if-moved" in arguments
    flags = [argument for argument in arguments
             if argument.startswith("--") and not argument.startswith("--call=") and argument != "--if-moved"]
    unknown = [world for world in named if world not in WORLDS]
    if flags or unknown or (moved and (named or calls)):
        sys.exit(f"{USAGE}; --if-moved records every world; not one: {' '.join(flags + unknown)}")
    if moved:
        was = STAMP.read_text().strip() if STAMP.is_file() else None
        now = key()
        if was == now and os.environ.get(FORCE) != "1":
            print(f"recordings: skipped, nothing they read or hold moved since the run of key {now} ({FORCE}=1 records)")
            sys.exit(0)
        STAMP.unlink(missing_ok=True)
    doors.binary = world_binary()
    print("world build:", doors.binary)
    if not named and not calls and out_root.is_dir():
        for folder in sorted(out_root.iterdir()):
            if folder.is_dir() and folder.name not in WORLDS:
                shutil.rmtree(folder)
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        for line in pool.map(lambda world: record(world, calls), named or list(WORLDS)):
            print(line)
    if moved:
        STAMP.parent.mkdir(parents=True, exist_ok=True)
        now = key()
        STAMP.write_text(now + "\n")
        print(f"recordings: recorded, key {now}")
