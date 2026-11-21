"""One door of the world build, `station guest serve --stdio` over a fresh copy of a world, shared by the recorder and
the probe: what one request changes, the next sees. `binary` is the world build it runs, set by the script that
imports it (world.py's world_binary).
"""

import json
import pathlib
import queue
import shutil
import subprocess
import sys
import threading
import time

from world import sandbox

QUIET = 10
UNENDED = {"overview.read"}
WHOLE = 30
binary = None
_guard = threading.Lock()
_locks, _turns = {}, {}


def callOf(request):
    return request.get("watch") or request.get("call")


def born(frames):
    """Whether a frame of an Overview is a birth's: a VM whose create still runs, answered from its Mac's record."""
    return any((frame.get("out") or {}).get("Creating") for frame in frames)


def never(heard):
    return []


def turned(seed, fixture):
    """The HOME of a world after its engines took one turn, made once per (seed, fixture) and copied by every door of
    that world, so each door owns a private copy and none sees another's writes."""
    key = (str(seed), fixture)
    with _guard:
        lock = _locks.setdefault(key, threading.Lock())
    with lock:
        if key not in _turns:
            home, env = sandbox(seed, fixture)
            # invariant: the world's engines take one turn before a door opens, as a Mac's engine has run before
            # the window does: what an engine keeps beside its loop (a VM's drift) is kept when it is asked. That
            # turn is the engine's life before the story, so the story's clock is given back as the seed holds it:
            # the turn's ticks move it, and the history the seed holds would read stale at the moment the window asks.
            subprocess.run([str(binary), "--version"], cwd=home, env=env, stdout=subprocess.DEVNULL, check=True)
            world = home / "world.json"
            kept = json.loads(world.read_text())
            kept["now"] = json.loads(pathlib.Path(seed).read_text())["now"]
            world.write_text(json.dumps(kept, indent=2) + "\n")
            _turns[key] = home
        return _turns[key]


class Door:
    """One `station guest serve --stdio` over a private copy of a world's turned HOME: what one request changes, the
    next sees."""

    def __init__(self, seed, fixture):
        self.home, env = sandbox(seed, fixture)
        self.env = env
        shutil.copytree(turned(seed, fixture), self.home, symlinks=True, dirs_exist_ok=True)
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

    def ask(self, requests, during=never):
        """Sends every request at once and returns, in their order, the answers of each, once all have ended. Each
        frame the first request answers is counted and handed to `during`, and the requests it returns are sent at
        once, while that one still runs, their answers returned after the others in the order they were sent. A watch
        of UNENDED the engine holds open, a VM's Overview through its birth (its last answer a `Creating`), is let go
        of after QUIET seconds of silence by a cancel frame, as the window lets go of it, and its answers end on the
        engine's done; any other is waited for to its end, as a watch that settles (disk.read) or an Overview whose
        slowest part has not answered may be silent longer than QUIET. A request still open after WHOLE silences in a
        row stops the run, so no part-read answer is ever written."""
        ids, watches, unended = [], set(), set()
        first = len(requests)
        unheld = False

        def sent(more):
            if not more:
                return
            fresh = [str(self.next + index) for index in range(len(more))]
            self.next += len(more)
            self.send([{"v": 2, "id": id, **request} for id, request in zip(fresh, more)])
            ids.extend(fresh)
            watches.update(id for id, request in zip(fresh, more) if "watch" in request)
            unended.update(id for id, request in zip(fresh, more) if "watch" in request and callOf(request) in UNENDED)

        sent(requests)
        answers, ended, cancelled, silences = {}, set(), set(), 0
        while len(ended) < len(ids):
            try:
                line = self.lines.get(timeout=QUIET)
            except queue.Empty:
                held = {id for id in unended - ended - cancelled if born(answers.get(id, [])[-1:])}
                if held:
                    self.send([{"v": 2, "id": id, "cancel": True} for id in sorted(held)])
                    cancelled |= held
                silences += 1
                if silences >= WHOLE:
                    self.close()
                    sys.exit(f"{len(ended)} of {len(ids)} requests ended after {WHOLE * QUIET} s of silence")
                continue
            silences = 0
            if not line:
                break
            frame = json.loads(line)
            ended_now = "hello" in frame or "error" in frame or "done" in frame
            ended_now = ended_now or ("out" in frame and frame["id"] not in watches)
            if ended_now:
                ended.add(frame["id"])
            answers.setdefault(frame["id"], []).append({key: value for key, value in frame.items() if key != "id"})
            if frame["id"] == ids[0]:
                sent(during(len(answers[ids[0]])))
            if not unheld and len(ids) > first and set(ids[first:]) <= ended and ids[0] not in ended:
                self.unhold()
                unheld = True
        if len(ended) < len(ids):
            self.close()
            sys.exit(f"{len(ended)} of {len(ids)} requests answered")
        return [answers[id] for id in ids]

    def reached(self):
        """Returns once every hold the world declares for a VM is reached, its call waiting: the world file lists it
        under `held`, written as that call lets the world go, so a read asked then reads the world where the hold
        keeps it, however loaded the run. With no such hold it returns at once; one not reached after WHOLE * QUIET
        seconds stops the run."""
        world = self.home / "world.json"
        for _ in range(WHOLE * QUIET * 50):
            try:
                kept = json.loads(world.read_text())
            except ValueError:
                kept = None
            if kept is not None:
                declared = [fault for fault in kept.get("script") or [] if fault.get("hold") and fault.get("vm")]
                if all(fault in (kept.get("held") or []) for fault in declared):
                    return
            time.sleep(0.02)
        self.close()
        sys.exit(f"a hold of the world was not reached after {WHOLE * QUIET} s")

    def unhold(self):
        """Lets go of every hold of the world, so a gesture the world holds goes on to its end once its `during`
        reads have answered; with nothing held it changes nothing."""
        subprocess.run([str(binary), "world", "unhold"], cwd=self.home, env=self.env, stdout=subprocess.DEVNULL,
                       check=True)

    def close(self):
        self.process.stdin.close()
        self.process.wait(timeout=60)
        shutil.rmtree(self.home, ignore_errors=True)
