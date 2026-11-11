"""value: a fake engine for the capture: one station.api/v2 socket that streams
one page to every page watch, holds every other watch open, and answers every
call with an empty out, engine.hello with the page revision the app reads.

usage: python3 replay.py <page.json> <socket> <page revision>
"""

import json
import os
import socket
import sys
import threading
import time


def send(conn, frame):
    conn.sendall((json.dumps(frame) + "\n").encode())


def serve(conn, page, revision):
    with conn:
        line = conn.makefile("rb").readline()
        try:
            request = json.loads(line)
        except ValueError:
            return
        try:
            if "call" in request:
                out = {"page": revision, "executable": "/usr/local/bin/station"} if request["call"] == "engine.hello" else {}
                send(conn, {"v": 2, "id": request.get("id", "1"), "out": out})
                return
            send(conn, {"v": 2, "id": request.get("id", "1"), "out": {"stream": True}})
            if request.get("watch") == "page":
                send(conn, {"v": 2, "id": request.get("id", "1"), "out": {"event": "page", "page": page}})
            while True:
                time.sleep(15)
                if request.get("watch") == "page":
                    send(conn, {"v": 2, "id": request.get("id", "1"), "out": {"event": "tick"}})
        except OSError:
            return


def main():
    page_path, path, revision = sys.argv[1], sys.argv[2], int(sys.argv[3])
    with open(page_path, encoding="utf-8") as f:
        page = json.load(f)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    if os.path.exists(path):
        os.unlink(path)
    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(path)
    server.listen(16)
    print(f"replay {page_path} on {path}", flush=True)
    while True:
        conn, _ = server.accept()
        threading.Thread(target=serve, args=(conn, page, revision), daemon=True).start()


if __name__ == "__main__":
    main()
