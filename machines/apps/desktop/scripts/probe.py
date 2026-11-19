"""Ask a seeded world what it answers, before a recording or a test leans on it.

Usage: probe.py <world> [--fault=<call>[:<times>]]... <request as JSON>..., in a Linux VM only: one door of the world
build over a fresh copy of the world as world.py seeds it (the recorder's Door), each request sent in order on that one
connection, so a gesture's change is read by the requests after it, and every frame each answers printed as one line
of JSON under the request. A `--fault` adds a line to the seed's `script`, so the fake's call fails that many times
(once by default), as a seed of WORLDS does (`first_refused`). Nothing is written under src/fixtures/: a state the
probe finds is made a recording by a line of WORLDS and of record.py's tables.
"""

import json
import sys

import record
from world import seeded, temporary, world_binary

USAGE = "usage: probe.py <world> [--fault=<call>[:<times>]]... <request as JSON>..."


def faulted(seed, faults):
    """A copy of the seed whose fake fails each call of `faults` its number of times."""
    if not faults:
        return seed
    content = json.loads(seed.read_text())
    content["script"] = [*content.get("script", []), *[
        {"call": call, "times": int(times or 1)} for call, _, times in (fault.partition(":") for fault in faults)]]
    copy = temporary("window-seed-") / seed.name
    copy.write_text(json.dumps(content))
    return copy


if __name__ == "__main__":
    arguments = sys.argv[1:]
    faults = [argument.removeprefix("--fault=") for argument in arguments if argument.startswith("--fault=")]
    rest = [argument for argument in arguments if not argument.startswith("--fault=")]
    if len(rest) < 2 or any(argument.startswith("--") for argument in rest):
        sys.exit(USAGE)
    world, requests = rest[0], [json.loads(argument) for argument in rest[1:]]
    seed, fixture = seeded(world)
    record.binary = world_binary()
    door = record.Door(faulted(seed, faults), fixture)
    for request in requests:
        [frames] = door.ask([request])
        print("request", json.dumps(request))
        for frame in frames:
            print(json.dumps(frame))
    door.close()
