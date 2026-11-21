"""value: the bench's reader, run on the Mac by run.sh --capture when the set holds
bench: it reads the app's bench log (Bench and OpeningLog in Trial.swift) and the
engine's log of each watch's opening (engine/process/opening.go), and prints the
p50 and p95 of each measure, then writes them to bench.json beside the logs.

usage: bench.py <capture dir>

Per opening of the panel (`open`, from the window shown) and of This Mac's page
over it (`push`, from the page opened):
  full     ms to its first full frame: the first from which the glass's height
           and the content generation no longer change, the page drawn with its
           header; none for an opening whose last frame is still a skeleton
  first    ms to its first logged frame
  gens     content generations drawn, the last included
  heights  times the glass's height changed
  skeleton frames drawn with a page still without its header
Per watch of the engine: first, header and full (ms after the watch opened),
frames before the full one, and the commands run meanwhile.

invariant: a bench that measured nothing fails: no opening of a kind, no opening of a
kind that reached a full frame, or no watch logged by the engine (an engine that
refused to start leaves the app on its skeleton, every opening one generation that
never moves).
"""

import json
import os
import re
import sys

FRAME = re.compile(r" (open|push) t=(\d+) frame=\d+ glass=x=\S+ y=\S+ w=\S+ h=([\d.]+) gen=(\S+)")
BENCH = re.compile(r" bench (open|push) (\d+)$")
WATCH = re.compile(r"watch: (\S+) first=(\S+) header=(\S+) full=(\S+) frames=(\d+) commands=(\d+)")


def pct(values, p):
    """The nearest-rank percentile; None for no value."""
    if not values:
        return None
    ordered = sorted(values)
    rank = max(1, -(-p * len(ordered) // 100))
    return ordered[int(rank) - 1]


def openings(log):
    """Each bench opening, by its kind, as its list of (t, height, generation)."""
    runs, current, kind = {"open": [], "push": []}, None, None
    for line in log.splitlines():
        m = BENCH.search(line)
        if m:
            kind, current = m.group(1), []
            runs[kind].append(current)
            continue
        f = FRAME.search(line)
        if f and current is not None and f.group(1) == kind:
            current.append((int(f.group(2)), float(f.group(3)), f.group(4)))
    return runs


def measure(frames):
    if not frames:
        return None
    last = frames[-1][1:]
    full = frames[-1][0]
    for t, h, g in reversed(frames):
        if (h, g) != last:
            break
        full = t
    if last[1].startswith("skeleton"):
        full = None
    gens = 1 + sum(1 for a, b in zip(frames, frames[1:]) if a[2] != b[2])
    heights = sum(1 for a, b in zip(frames, frames[1:]) if a[1] != b[1])
    skeleton = sum(1 for f in frames if f[2].startswith("skeleton"))
    return {"full": full, "first": frames[0][0], "gens": gens, "heights": heights, "skeleton": skeleton}


def ms(word):
    return None if word == "-" else int(word.rstrip("ms"))


def watches(log):
    out = {}
    for m in WATCH.finditer(log):
        page = m.group(1).split(":")[0]
        out.setdefault(page, []).append({"first": ms(m.group(2)), "header": ms(m.group(3)), "full": ms(m.group(4)),
                                         "frames": int(m.group(5)), "commands": int(m.group(6))})
    return out


def summary(rows, keys):
    out = {"n": len(rows)}
    for k in keys:
        values = [r[k] for r in rows if r.get(k) is not None]
        out[k] = {"p50": pct(values, 50), "p95": pct(values, 95), "max": max(values) if values else None, "n": len(values)}
    return out


def bench(app, engine):
    result = {}
    for kind, runs in openings(app).items():
        rows = [r for r in (measure(f) for f in runs) if r]
        result[kind] = summary(rows, ["full", "first", "gens", "heights", "skeleton"])
    for page, rows in watches(engine).items():
        result[f"watch {page}"] = summary(rows, ["first", "header", "full", "frames", "commands"])
    return result


def refusals(result):
    """Why the bench measured nothing, one reason per line; none for a bench that measured."""
    out = []
    for kind in ("open", "push"):
        if not result.get(kind, {}).get("n"):
            out.append(f"no {kind} opening measured")
        elif not result[kind]["full"]["n"]:
            out.append(f"no {kind} opening reached a full frame")
    if not any(name.startswith("watch ") for name in result):
        out.append("the engine logged no watch")
    return out


def main():
    folder = sys.argv[1]
    app = open(os.path.join(folder, "bench-app.log"), encoding="utf-8").read()
    engine_path = os.path.join(folder, "bench-engine.log")
    engine = open(engine_path, encoding="utf-8").read() if os.path.exists(engine_path) else ""
    result = bench(app, engine)
    json.dump(result, open(os.path.join(folder, "bench.json"), "w", encoding="utf-8"), indent=1)
    for name, s in result.items():
        cells = " ".join(f"{k} p50={v['p50']} p95={v['p95']} max={v['max']}" for k, v in s.items() if k != "n")
        print(f"bench {name} n={s['n']}: {cells}")
    reasons = refusals(result)
    for reason in reasons:
        print(f"bench: {reason}")
    sys.exit(1 if reasons else 0)


if __name__ == "__main__":
    main()
