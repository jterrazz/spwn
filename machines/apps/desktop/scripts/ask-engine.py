"""One call to an engine's socket on a connection of its own, as another client of that engine would make it.

Usage: ask-engine.py <socket> <call> [<input as JSON>]; in a Linux VM only, against serve-world.py's socket, so a
live session changes the world beside the window (`machines.stop`) and sees whether the window follows. It prints the
call's last frame, its `out` named by its keys only or its error's type and message, and exits 1 on an error or a
connection that ends unanswered. A world build's CLI run beside the door is no such client: the door holds its world
from its start and writes it back at its end (`apps/engine/wire/world_on.go`, `worldRun`), so the door never reads its change.
"""

import json
import socket
import sys

path, call = sys.argv[1], sys.argv[2]
given = json.loads(sys.argv[3]) if len(sys.argv) > 3 else {}
connection = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
connection.settimeout(120)
connection.connect(path)
connection.sendall(json.dumps({"v": 2, "id": "ask", "call": call, "in": given}).encode() + b"\n")
pending = b""
while data := connection.recv(1 << 20):
    pending += data
    *lines, pending = pending.split(b"\n")
    for line in lines:
        frame = json.loads(line)
        if "error" in frame:
            print(call, "error", frame["error"].get("type", ""), frame["error"].get("message", ""))
            sys.exit(1)
        if "out" in frame:
            out = frame["out"]
            print(call, "out", sorted(out) if isinstance(out, dict) else type(out).__name__)
            sys.exit(0)
print(call, "the engine closed the connection before it answered")
sys.exit(1)
