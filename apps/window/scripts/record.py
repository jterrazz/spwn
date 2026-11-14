"""Record the engine's real answers the window's pages read, over the worlds of specs/usecases/worlds/.

Usage: record.py [--gestures] [--call=<call>...] [--only=<call> [--in=<json>]] [<world>...], every world already
recorded and every world of SEEDS
when none is named, in a Linux VM only: it asks `station guest serve --stdio` of the world build (world.py), the door
specs/architecture/answers_test.go asks, over a fresh copy of each world in a temporary HOME (one per request,
so no answer depends on what another request did to the world), and writes one file per call:
apps/window/src/fixtures/worlds/<world>/<call>.json, every request of that call the pages make in that world,
with its answers. A page that reads a new call, or the same call with a new input, adds its requests to ASKS
(or FOLLOWS, when they depend on another answer) and records again the worlds its specs use.

A gesture of GESTURES is recorded in a fresh world of its own: the gesture, then every read of that world asked
again in the same door, one after the other, with the reads of a machine the gesture created, and written to
<world>/gestures/<call>.json with the reads whose answer it changed; the replay transport plays it, then answers
those reads as they were after it. `--gestures` records the gestures alone, over the reads already recorded;
`--call=<call>` records only that call's gestures, in the worlds named or every world GESTURES gives it, and rewrites
no other file. `--only=<call>` asks again the reads of that one call, in the worlds named or every world that records
it, and splices their answers into <world>/<call>.json in place; `--in=<json>` narrows it to the requests of that
input (of every target), or adds that request when none is recorded yet. No other exchange and no other file moves.

The world build's clock moves when a watch waits on it (apps/engine/wire/world_on.go), so two runs do not write the same
files: a watch the engine ends before a check settles (accounts.list, disk.read) answers pending in one run and
settled in the other. The recorder keeps of a watch its first answer and its last (what streams between comes in the
scheduler's order), never counts a read of TIMED as changed by a gesture, and keeps the NOISE field of a gesture's
last recording: the process number the host gave, which no world decides, and the CLOCKED field of an answer (settings.matrix's Read, the world's clock) as its last
recording holds it.

A far Mac's machine, asked through that door, is answered with This Mac's overview: only This Mac, with no
target, and its VMs, by name, are asked. The worlds of world.py's SEEDS are recorded too.
"""

import concurrent.futures
import json
import queue
import shutil
import subprocess
import sys
import threading

from world import ROOT, SEEDS, sandbox, seeded, world_binary

out_root = ROOT / "apps" / "window" / "src" / "fixtures" / "worlds"
GRAPHS = (360 * 10**9, 3600 * 10**9, 86400 * 10**9)
SETTLED = {"apps.list", "settings.list"}
TIMED = {"activity.read", "engine.watch"}
NOISE = {"engine.start": "PID"}
CLOCKED = {"settings.matrix": "Read"}
PRIOR = {}
MACHINES = {"watch": "machines.list", "in": {}}
QUIET = 10
UNENDED = {"overview.read"}
binary = world_binary()


class Door:
    """One `station guest serve --stdio` over a fresh copy of a world: what one request changes, the next sees."""

    def __init__(self, seed, fixture):
        self.home, env = sandbox(seed, fixture)
        self.process = subprocess.Popen(
            [str(binary), "guest", "serve", "--stdio"], cwd=self.home, env=env,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True,
        )
        self.lines = queue.Queue()
        threading.Thread(target=self.pump, daemon=True).start()
        self.next = 0
        self.station = self.ask([{"hello": {"version": "dev", "calls": None}}])[0][0]["hello"]["version"]

    def pump(self):
        for line in self.process.stdout:
            self.lines.put(line)
        self.lines.put("")

    def send(self, frames):
        self.process.stdin.write("".join(json.dumps(frame) + "\n" for frame in frames))
        self.process.stdin.flush()

    def ask(self, requests):
        """Sends every request at once and returns, in their order, the answers of each, once all have ended. A watch
        of UNENDED the engine holds open (a VM's Overview through its birth) is let go of after QUIET seconds of
        silence by a cancel frame, as the window lets go of it, and its answers end on the engine's done; any other
        is waited for to its end, as a watch that settles (accounts.list) may be silent longer than QUIET."""
        ids = [str(self.next + index) for index in range(len(requests))]
        self.next += len(requests)
        self.send([{"v": 2, "id": id, **request} for id, request in zip(ids, requests)])
        watches = {id for id, request in zip(ids, requests) if "watch" in request}
        unended = {id for id, request in zip(ids, requests) if "watch" in request and callOf(request) in UNENDED}
        answers, ended, cancelled = {}, set(), set()
        while len(ended) < len(ids):
            try:
                line = self.lines.get(timeout=QUIET)
            except queue.Empty:
                held = unended - ended - cancelled
                if held:
                    self.send([{"v": 2, "id": id, "cancel": True} for id in sorted(held)])
                    cancelled |= held
                continue
            if not line:
                break
            frame = json.loads(line)
            ended_now = "hello" in frame or "error" in frame or "done" in frame
            ended_now = ended_now or ("out" in frame and frame["id"] not in watches)
            if ended_now:
                ended.add(frame["id"])
            answers.setdefault(frame["id"], []).append({key: value for key, value in frame.items() if key != "id"})
        if len(ended) < len(ids):
            self.close()
            sys.exit(f"{len(ended)} of {len(ids)} requests answered")
        return [answers[id] for id in ids]

    def close(self):
        self.process.stdin.close()
        self.process.wait(timeout=60)
        shutil.rmtree(self.home, ignore_errors=True)


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


def asks(listed):
    """The reads the window asks of this Mac and its VMs. This Mac's engine is the socket's own, so a read of it
    names no target, as targetOf (src/model/machines.ts) sends it, and a read two pages ask is asked once. A drift
    names its machine in its input and no target, as driftAsked sends it. The New machine sheet watches machines.new
    of the socket's engine, opened anywhere or on one Mac machines.list lists, as askedOf sends it; a sheet opened
    from a machine file names it in From, a request `--only=machines.new --in=<json>` adds."""
    here = thisMac(listed)
    vms = [(mac, vm) for mac in here for vm in mac.get("VMs") or []]
    targets = [{} for _ in here] + [at(vm["Name"]) for _, vm in vms]
    return list({keyOf(request): request for request in [
        {"watch": "overview.read", "in": {}},
        {"call": "engine.read", "in": {}},
        {"call": "ports.list", "in": {"AllMachines": True}},
        {"call": "images.list", "in": {}},
        {"watch": "apps.list", "in": {}},
        {"watch": "settings.list", "in": {}},
        {"call": "settings.matrix", "in": {}},
        {"call": "machinefile.explain", "in": {}},
        {"watch": "engine.watch", "in": {"machines": []}},
        *[{"watch": "engine.watch", "in": {"machines": [vm["Name"]]}} for _, vm in vms],
        *[{kind: call, "in": {}, **target} for target in targets for kind, call in (
            ("watch", "overview.read"), ("watch", "apps.list"), ("watch", "settings.list"),
            ("call", "machinefile.plan"), ("call", "machinefile.explain"), ("watch", "disk.read"),
        )],
        {"call": "machinefile.drift", "in": {}},
        *[{"call": "machinefile.drift", "in": {"Machine": vm["Name"]}} for _, vm in vms],
        *[{"call": call, "in": {"Machinefile": "station-macos"}}
          for _ in here for call in ("machinefile.drift", "machinefile.plan")],
        *[{"watch": "activity.read", "in": {"graph": graph}, **target} for target in targets for graph in GRAPHS],
        *[{"watch": "activity.read", "in": {"graph": graph, "of": vm["Name"], "machine": {"Name": vm["Name"], "On": mac["Name"]}}}
          for mac, vm in vms for graph in GRAPHS],
        *[{"watch": "activity.read", "in": {"history": True}, **target} for target in targets],
        *[{"watch": "activity.read", "in": {"history": True, "of": vm["Name"], "machine": {"Name": vm["Name"], "On": mac["Name"]}}}
          for mac, vm in vms],
        *[{"call": "machines.log", "in": {"VM": vm["Name"]}} for _, vm in vms],
        *[{"call": "machines.ssh", "in": {"Name": vm["Name"]}} for _, vm in vms],
        *[{"watch": "accounts.list", "in": {"VMs": True}} for _ in here],
        *[{"watch": "awake.read", "in": {}} for _ in here],
        {"watch": "machines.new", "in": {}},
        *[{"watch": "machines.new", "in": {"On": mac}} for mac in macsOf(listed)],
    ]}.values())


def follows(requests, answered):
    explained = [last_out(frames) for request, frames in zip(requests, answered)
                 if request.get("call") == "machinefile.explain" and "target" not in request]
    names = [file["Name"] for out in explained if out for file in out.get("Files") or [] if file.get("Name")]
    asked_already = {keyOf(request) for request in requests}
    drifts = [{"call": "machinefile.drift", "in": {"Machinefile": name}} for name in dict.fromkeys(names)]
    return [drift for drift in drifts if keyOf(drift) not in asked_already]


def callOf(request):
    return request.get("watch") or request.get("call")


def asked(seed, requests, fixture):
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        return list(pool.map(lambda request: serve(seed, [request], fixture)[1][0], requests))


GESTURES = {
    "two-macs": [
        {"call": "machines.stop", "in": {"VM": "linux-1", "Confirm": "yes"}},
        {"call": "machines.start", "in": {"VM": "api"}},
        {"call": "machines.delete", "in": {"VM": "api", "Confirm": "api"}},
        {"call": "machines.restart", "in": {"Name": "infra", "Confirm": "yes"}, **at("jterrazz-mac-mini")},
        {"call": "apps.update", "in": {"Managers": ["mise"]}},
        {"call": "disk.free", "in": {}},
        {"call": "disk.free", "in": {"Apps": ["go"]}},
        {"call": "disk.free", "in": {}, **at("linux-1")},
        {"call": "disk.move", "in": {"Folder": "~/.tart"}},
        {"call": "disk.erase", "in": {"App": "ollama", "Confirm": "ollama"}},
        {"call": "disk.erase", "in": {"App": "orbstack", "Confirm": "orbstack"}},
        {"call": "images.delete", "in": {"Image": "station-linux", "Version": "2026.10.20", "Confirm": "yes"}},
        {"call": "ports.stop-sharing", "in": {"VM": "linux-1", "Port": "5173", "Confirm": "yes"}},
        {"call": "accounts.share", "in": {"Account": "github", "With": "linux-1", "For": 12 * 3600 * 10**9}},
        {"call": "accounts.take-back", "in": {"Account": "github", "From": "linux-1", "Confirm": "yes"}},
        {"call": "background.remove", "in": {"Label": "com.jterrazz.brain.x-snapshot", "Confirm": "yes"}},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "station-linux", "Lifetime": "persistent", "Size": {"MemoryBytes": 8589934592}}, "Confirm": "yes"}, **at("jterrazz-mac-mini")},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "station-linux", "Lifetime": "ephemeral"}, "Confirm": "yes"}},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "linux-1", "From": "station-linux", "Lifetime": "persistent"}, "Confirm": "yes"}, **at("jterrazz-mac-mini")},
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "linux-2", "From": "station-linux", "Lifetime": "persistent", "Purpose": "x" * 121}, "Confirm": "yes"}, **at("jterrazz-mac-mini")},
        {"call": "machines.create-server", "in": {"Name": "web", "Image": "station-linux", "Size": {"MemoryBytes": 4294967296}, "Confirm": "yes"}},
        {"call": "machines.add", "in": {"Name": "jterrazz-mac-mini"}},
    ],
    "stories": [
        {"call": "ports.share", "in": {"VM": "linux-1", "Port": "4321"}},
        {"call": "ports.share", "in": {"VM": "linux-3", "Port": "5432"}, **at("mac-mini")},
        {"call": "accounts.share", "in": {"Account": "claude", "With": "linux-1", "For": 12 * 3600 * 10**9}},
        {"call": "ports.declare", "in": {"Machine": "", "Port": "5000", "Via": "lan"}},
    ],
    "engine-stopped": [
        {"call": "machines.stop", "in": {"VM": "linux-1", "Confirm": "yes"}},
        {"call": "engine.start", "in": {}},
    ],
    "one-mac": [
        {"call": "machines.create-workspace", "in": {"Request": {"Name": "gate5", "From": "station-linux", "Lifetime": "persistent"}, "Confirm": "yes"}},
        {"call": "machines.add", "in": {"Name": "jterrazz-mac-mini"}},
    ],
    "creating": [
        {"call": "machines.delete", "in": {"VM": "linux-5", "Confirm": "linux-5"}},
    ],
    "machinefile-differs": [
        {"call": "machinefile.apply", "in": {"Confirm": "yes"}},
        {"call": "machinefile.apply", "in": {"Confirm": "yes"}, **at("api")},
        {"call": "background.remove", "in": {"Label": "com.jterrazz.brain.x-snapshot", "Confirm": "yes"}},
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
}


def bounded(frames):
    outs = [index for index, frame in enumerate(frames) if "out" in frame]
    ends = {outs[0], outs[-1]} if outs else set()
    return [frame for index, frame in enumerate(frames) if "out" not in frame or index in ends]


def kept(request, frames):
    if "watch" not in request:
        return frames
    return settled(frames) if callOf(request) in SETTLED else bounded(frames)


def keyOf(request):
    return json.dumps(request, sort_keys=True)


def thisMac(listed):
    out = last_out(listed) or {}
    return [mac for mac in (out.get("All") or {}).get("Macs") or [] if mac.get("ThisMac")]


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


def steadied(path, key, items):
    """An answer of CLOCKED keeps the clock field its last recording holds, so a recording again moves no file for
    time that passed."""
    was = PRIOR.get(path) or (json.loads(path.read_text()) if path.exists() else {})
    held = {keyOf(item["request"]): item for item in was.get(key, [])}
    for item in items:
        before = held.get(keyOf(item["request"]))
        if not before:
            continue
        field = CLOCKED.get(callOf(item["request"]))
        if field:
            carry(item["answers"], before["answers"], field)
        reads = {keyOf(read["request"]): read for read in before.get("after", [])}
        for read in item.get("after", []):
            field, one = CLOCKED.get(callOf(read["request"])), reads.get(keyOf(read["request"]))
            if field and one:
                carry(read["answers"], one["answers"], field)


def write(path, world, station, key, items):
    steadied(path, key, items)
    head = json.dumps({"world": world, "station": station})[:-1]
    lines = ",\n".join(json.dumps(item, separators=(",", ":")) for item in items)
    rest = '"exchanges": [], "gestures"' if key == "gestures" else '"exchanges"'
    path.write_text(f'{head}, {rest}: [\n{lines}\n]}}\n')


def gesture(seed, fixture, request, reads, before):
    """The gesture in a fresh world, then every read in that same world, kept when its answer changed: a watch by
    its settled answer, the one a window shows once it has read again. A read no recording holds yet, of a machine
    the gesture made, is kept whatever it answers."""
    door = Door(seed, fixture)
    [answers] = door.ask([request])
    [listed] = door.ask([MACHINES])
    born = [read for read in asks(listed) if keyOf(read) not in before]
    changed = []
    for read in [*reads, *born]:
        if callOf(read) in TIMED:
            continue
        [frames] = door.ask([read])
        frames, was = kept(read, frames), before.get(keyOf(read))
        if was is None or (settled(frames) != settled(was) if "watch" in read else frames != was):
            changed.append({"request": read, "answers": frames})
    door.close()
    return {"request": request, "answers": answers, "after": changed}


def recorded(folder):
    """The requests a world's recording holds and their answers, read back from its files."""
    requests, before = [], {}
    for path in sorted(folder.glob("*.json")):
        for exchange in json.loads(path.read_text())["exchanges"]:
            requests.append(exchange["request"])
            before[keyOf(exchange["request"])] = exchange["answers"]
    return requests, before


def steady(item, path):
    """A gesture's answer keeps the NOISE field its last recording holds: the number the host gave a process."""
    field = NOISE.get(callOf(item["request"]))
    if not field or not path.exists():
        return
    was = [one for one in json.loads(path.read_text())["gestures"] if one["request"] == item["request"]]
    held = [frame["out"][field] for one in was for frame in one["answers"] if field in frame.get("out", {})]
    for frame in item["answers"]:
        if held and field in frame.get("out", {}):
            frame["out"][field] = held[0]


def gestures(world, seed, fixture, folder, requests, before):
    chosen = [request for request in GESTURES.get(world, []) if not calls or callOf(request) in calls]
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        made = list(pool.map(lambda request: gesture(seed, fixture, request, requests, before), chosen))
    by_gesture = {}
    for request, item in zip(chosen, made):
        by_gesture.setdefault(callOf(request), []).append(item)
    for item in made:
        steady(item, folder / "gestures" / f"{callOf(item['request'])}.json")
    if not calls:
        shutil.rmtree(folder / "gestures", ignore_errors=True)
    if by_gesture:
        (folder / "gestures").mkdir(exist_ok=True)
    station = json.loads(next(folder.glob("*.json")).read_text())["station"]
    for call, items in by_gesture.items():
        write(folder / "gestures" / f"{call}.json", world, station, "gestures", items)
    return sum(len(items) for items in by_gesture.values())


def record(world):
    seed, fixture = seeded(world)
    folder = out_root / world
    if alone:
        requests, before = recorded(folder)
        made = gestures(world, seed, fixture, folder, requests, before)
        return f"{world}: {made} gestures over {len(requests)} recorded requests -> {folder}"
    station, [listed] = serve(seed, [MACHINES], fixture)
    requests = asks(listed)
    answered = asked(seed, requests, fixture)
    more = follows(requests, answered)
    if more:
        requests, answered = requests + more, answered + asked(seed, more, fixture)
    requests, answered = [MACHINES, *requests], [listed, *answered]
    by_call, before = {}, {}
    for request, frames in zip(requests, answered):
        by_call.setdefault(callOf(request), []).append({"request": request, "answers": kept(request, frames)})
        before[keyOf(request)] = kept(request, frames)
    folder.mkdir(parents=True, exist_ok=True)
    PRIOR.update({path: json.loads(path.read_text()) for path in folder.rglob("*.json")})
    for stale in folder.glob("*.json"):
        stale.unlink()
    for call, exchanges in by_call.items():
        write(folder / f"{call}.json", world, station, "exchanges", exchanges)
    made = gestures(world, seed, fixture, folder, requests, before)
    return f"{world}: {len(requests)} requests in {len(by_call)} calls, {made} gestures -> {folder}"


def again(world):
    """The exchanges of one read call asked again, those of the input given alone when one is, and spliced into
    the world's file of that call in place: every other exchange, and every other file, is left as recorded."""
    seed, fixture = seeded(world)
    path = out_root / world / f"{only}.json"
    held = json.loads(path.read_text())["exchanges"] if path.exists() else []
    station, [listed] = serve(seed, [MACHINES], fixture)
    known = [*[exchange["request"] for exchange in held], *asks(listed)]
    of_call = [request for request in known if callOf(request) == only]
    wanted = list({keyOf(request): request for request in of_call
                   if given is None or request["in"] == given}.values())
    if given is not None and not wanted:
        if not of_call:
            sys.exit(f"{world}: no recording nor ASKS names {only}, so its kind (call or watch) is unknown")
        kind = "watch" if "watch" in of_call[0] else "call"
        wanted = [{kind: only, "in": given}]
    fresh = {keyOf(request): {"request": request, "answers": kept(request, frames)}
             for request, frames in zip(wanted, asked(seed, wanted, fixture))}
    exchanges = [fresh.pop(keyOf(exchange["request"]), exchange) for exchange in held] + list(fresh.values())
    write(path, world, station, "exchanges", exchanges)
    return f"{world}: {len(wanted)} exchanges of {only} asked again -> {path}"


if __name__ == "__main__":
    print("world build:", binary)
    arguments = sys.argv[1:]
    calls = {argument.removeprefix("--call=") for argument in arguments if argument.startswith("--call=")}
    only = next((argument.removeprefix("--only=") for argument in arguments if argument.startswith("--only=")), None)
    given = next((json.loads(argument.removeprefix("--in=")) for argument in arguments if argument.startswith("--in=")), None)
    alone = "--gestures" in arguments or bool(calls)
    named = [argument for argument in arguments if not argument.startswith("--")]
    everywhere = sorted({folder.name for folder in out_root.iterdir() if folder.is_dir()} | set(SEEDS))
    if only is not None:
        worlds = named or [world for world in everywhere if (out_root / world / f"{only}.json").exists()]
        job = again
    else:
        worlds = named or [world for world in everywhere if not calls or calls & {callOf(it) for it in GESTURES.get(world, [])}]
        job = record
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        for line in pool.map(job, worlds):
            print(line)
