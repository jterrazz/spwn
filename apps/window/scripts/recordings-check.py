"""Replay every recording of every world against the world build of this tree, and fail on one the engine no longer
answers so.

Usage: recordings-check.py [<world>...], every recorded world when none is named, in a Linux VM only. Each exchange of
<world>/<call>.json is asked again in a fresh door, as record.py asks it, and each gesture of <world>/gestures/ is made
again by record.py's own `gesture`, in a fresh world, every read asked after it in the recorder's order. An answer is compared once reduced
as the recorder keeps it: a watch by its settled answer, a gesture's NOISE field and the CLOCKED field of a call left out, and a gesture's reads of
RACING left out, the watches whose settling races the engine's end. A leaf that differs is asked
once more, and a leaf two fresh answers disagree on is the world's clock, never a meaning: it is ignored. So a recording
is stale when the engine of this tree gives another answer, every time: a word, a field, a refusal it no longer gives.

Each stale file is printed with the request and the first paths that differ, then a last line,
`recordings: <n> stale of <m> files`, and the exit is 1 when any is.
"""

import concurrent.futures
import functools
import json
import sys
import time

from record import NOISE, Door, callOf, gesture, keyOf, kept, out_root, recorded, seeded, settled, unclocked

MISSING = object()
RACING = {"accounts.list", "disk.read"}
SHOWN = 4


def reduced(request, frames):
    frames = kept(request, frames)
    frames = settled(frames) if "watch" in request else frames
    frames = unclocked(request, frames)
    field = NOISE.get(callOf(request))
    if not field:
        return frames
    return [{**frame, "out": {key: value for key, value in frame["out"].items() if key != field}}
            if isinstance(frame.get("out"), dict) else frame for frame in frames]


def leaves(value, path=""):
    if isinstance(value, dict) and value:
        return {leaf: one for key, item in value.items() for leaf, one in leaves(item, f"{path}.{key}").items()}
    if isinstance(value, list) and value:
        return {leaf: one for index, item in enumerate(value) for leaf, one in leaves(item, f"{path}[{index}]").items()}
    return {path: value}


def differing(one, other):
    mine, theirs = leaves(one), leaves(other)
    return {path for path in mine.keys() | theirs.keys() if mine.get(path, MISSING) != theirs.get(path, MISSING)}


@functools.cache
def seed_of(world):
    return seeded(world)


def read_now(world, request):
    seed, fixture = seed_of(world)
    door = Door(seed, fixture)
    [frames] = door.ask([request])
    door.close()
    return reduced(request, frames)


@functools.cache
def reads_of(world):
    return recorded(out_root / world)


def gesture_shown(item):
    return {"answers": reduced(item["request"], item["answers"]),
            "after": {keyOf(read["request"]): reduced(read["request"], read["answers"]) for read in item["after"]}}


def gesture_now(world, item):
    """The gesture made again as record.py makes it, every read asked after it in the same order, so the world's clock
    reads the same at each answer."""
    seed, fixture = seed_of(world)
    requests, before = reads_of(world)
    return gesture_shown(gesture(seed, fixture, item["request"], requests, before))


def racing(key):
    return callOf(json.loads(key)) in RACING


def beside(one, other):
    """A gesture's two readings with its reads of RACING left out of both: a watch the engine ends before a check
    settles answers pending in one run and settled in the other (scripts/README.md, What a recording holds)."""
    if not isinstance(one, dict) or "after" not in one:
        return one, other
    return tuple({**side, "after": {key: frames for key, frames in side["after"].items() if not racing(key)}}
                 for side in (one, other))


def judged(was, ask):
    """The paths where the engine answers otherwise than the recording, those its own two answers disagree on aside."""
    now = ask()
    paths = differing(*beside(was, now))
    return sorted(paths - differing(*beside(now, ask()))) if paths else []


def exchange_job(world, file, exchange):
    request = exchange["request"]
    paths = judged(reduced(request, exchange["answers"]), lambda: read_now(world, request))
    return file, request, paths


def gesture_job(world, file, item):
    return file, item["request"], judged(gesture_shown(item), lambda: gesture_now(world, item))


def jobs(world):
    folder = out_root / world
    for path in sorted(folder.glob("*.json")):
        for exchange in json.loads(path.read_text())["exchanges"]:
            yield exchange_job, world, path.relative_to(out_root), exchange
    for path in sorted((folder / "gestures").glob("*.json")):
        for item in json.loads(path.read_text())["gestures"]:
            yield gesture_job, world, path.relative_to(out_root), item


def main():
    started = time.monotonic()
    worlds = sys.argv[1:] or sorted(folder.name for folder in out_root.iterdir() if folder.is_dir())
    work = [job for world in worlds for job in jobs(world)]
    files, stale = {str(file) for _, _, file, _ in work}, {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        for file, request, paths in pool.map(lambda job: job[0](*job[1:]), work):
            if paths:
                stale.setdefault(str(file), []).append((request, paths))
    for file, found in sorted(stale.items()):
        print(f"{file}: {len(found)} stale")
        for request, paths in found:
            more = f" and {len(paths) - SHOWN} more" if len(paths) > SHOWN else ""
            print(f"  {json.dumps(request, sort_keys=True)}: {', '.join(paths[:SHOWN])}{more}")
    print(f"recordings: {len(stale)} stale of {len(files)} files, {len(work)} recordings replayed"
          f" · {round(time.monotonic() - started)} s")
    return 1 if stale else 0


if __name__ == "__main__":
    sys.exit(main())
