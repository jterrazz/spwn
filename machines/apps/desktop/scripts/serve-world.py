"""The engine's socket over a world: a unix socket whose every connection speaks to one `station guest serve --stdio`.

Usage: serve-world.py [<world> [<socket>]], stories and <STATION_BENCH or the temp folder>/run/<world>.sock by
default; in a Linux VM only, for the Tauri window to read a world as it reads a Mac's engine
(STATION_ENGINE_SOCKET=<socket>). One door serves every connection over one copy of the world in a sandbox of its
own (world.py), as one engine serves every client: what a gesture changes, the next read sees, until the bridge
ends. Each connection's request ids are renamed `<connection>:<id>` on their way in and back on their way out, so
two windows' ids never meet. The door wants a hello first: the bridge sends its own and swallows its answer.
It prints one line per request it bridges (its time, `call` or `watch`, the call's name, the renamed id) and one per
answer to `engine.read` (the State and Version answered, or the error's type): names and states only, never an input
or an answer's other values, so the log holds no credential.
"""

import json
import os
import pathlib
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import threading
from datetime import datetime

from world import sandbox, seeded, world_binary

world = sys.argv[1] if len(sys.argv) > 1 else "stories"
bench = pathlib.Path(os.environ.get("STATION_BENCH") or tempfile.gettempdir())
path = sys.argv[2] if len(sys.argv) > 2 else str(bench / "run" / f"{world}.sock")
seed, fixture = seeded(world)
home, env = sandbox(seed, fixture)
door = subprocess.Popen(
    [str(world_binary()), "guest", "serve", "--stdio"], cwd=home, env=env,
    stdin=subprocess.PIPE, stdout=subprocess.PIPE, bufsize=0,
)
door.stdin.write(b'{"v":2,"id":"bridge-hello","hello":{"version":"dev","calls":null}}\n')
lock = threading.Lock()
connections = {}
asked = {}


def log(*words):
    print(datetime.now().strftime("%H:%M:%S.%f")[:-3], *words, flush=True)


def answered(frame):
    name = asked.get(frame.get("id"))
    if name is None or ("out" not in frame and "error" not in frame):
        return
    asked.pop(frame.get("id"), None)
    if name != "engine.read":
        return
    if "error" in frame:
        log("answer", name, frame["id"], "error", (frame["error"] or {}).get("type", ""))
        return
    out = frame["out"] or {}
    state, version = out.get("State", ""), out.get("Version", "")
    log("answer", name, frame["id"], f"State={state}", f"Version={version!r}")


def down():
    for line in door.stdout:
        frame = json.loads(line)
        answered(frame)
        number, _, id = str(frame.get("id", "")).partition(":")
        conn = connections.get(number)
        if conn is None:
            continue
        frame["id"] = id
        try:
            conn.sendall(json.dumps(frame).encode() + b"\n")
        except OSError:
            connections.pop(number, None)


def handle(number, conn):
    connections[number] = conn
    pending = b""
    try:
        while data := conn.recv(65536):
            pending += data
            *lines, pending = pending.split(b"\n")
            for line in lines:
                if line.strip():
                    frame = json.loads(line)
                    frame["id"] = f"{number}:{frame.get('id', '')}"
                    name = frame.get("call") or frame.get("watch")
                    if name:
                        asked[frame["id"]] = name
                        log("call" if "call" in frame else "watch", name, frame["id"])
                    with lock:
                        door.stdin.write(json.dumps(frame).encode() + b"\n")
    except OSError:
        pass
    connections.pop(number, None)
    conn.close()


threading.Thread(target=down, daemon=True).start()
pathlib.Path(path).parent.mkdir(parents=True, exist_ok=True)
if os.path.exists(path):
    os.unlink(path)
signal.signal(signal.SIGTERM, lambda *_: sys.exit(0))
server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
server.bind(path)
server.listen(16)
print("serving", world, "on", path, flush=True)
count = 0
try:
    while True:
        connection, _ = server.accept()
        count += 1
        threading.Thread(target=handle, args=(str(count), connection), daemon=True).start()
finally:
    door.stdin.close()
    door.wait(timeout=60)
    shutil.rmtree(home, ignore_errors=True)
