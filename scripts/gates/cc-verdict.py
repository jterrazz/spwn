#!/usr/bin/env python3
"""cc-verdict.py: one verdict per cc commit block: GREEN, FLAKE(<name>) or RED.

usage:
  cc-verdict.py --block <file> [--patch <drift.patch>] [--only <STEP>]   one commit's output; prints "<VERDICT> steps=<failed steps> [why]"
  cc-verdict.py --replay <cc log>...                                     every "=== n sha" block of old logs (no patch: drift flakes match on their file only)
Steps: GOFMT VET LINT GEN DRIFT GOTEST MAKELINT BUILD STORIES; INCOMPLETE when the block never reached CC-DONE.
--only <STEP> judges one step alone (a flake rerun's output). A FLAKE names every signature that explains every failed step;
a failed step no signature explains, or a drift whose diff shape differs from its signature's, is RED.
Exit code: 0 for GREEN and FLAKE, 1 for RED (--block only).
"""
import json
import re
import sys

LOAD_CURVE = "usecases/activity/read/a-load-curve-moves-with-the-world-clock.spec.yaml"
CTRL_C = "usecases/errors/interrupt/silent.spec.yaml"
W = "apps/desktop/src/fixtures/worlds/"
UP_NS = 200_000_000


def jdiff(a, b, path=""):
    if type(a) is not type(b):
        yield path, a, b
    elif isinstance(a, dict):
        for k in sorted(set(a) | set(b)):
            if k not in a or k not in b:
                yield path + "/" + k, a.get(k, "<absent>"), b.get(k, "<absent>")
            else:
                yield from jdiff(a[k], b[k], path + "/" + k)
    elif isinstance(a, list):
        if len(a) != len(b):
            yield path, a, b
        else:
            for i, (x, y) in enumerate(zip(a, b)):
                yield from jdiff(x, y, f"{path}[{i}]")
    elif a != b:
        yield path, a, b


def shape_overview(old, new):
    for path, a, b in jdiff(old, new):
        if path.endswith("/Readiness") and {repr(a), repr(b)} == {"[]", "None"}:
            continue
        if isinstance(a, str) and isinstance(b, str) and {("context canceled" in a), ("context canceled" in b)} == {True, False} and ("guest_absent" in a or "guest_absent" in b):
            continue
        return False
    return True


def shape_up(old, new):
    for path, a, b in jdiff(old, new):
        if path.endswith("/Up") and isinstance(a, (int, float)) and isinstance(b, (int, float)) and abs(a - b) <= UP_NS:
            continue
        return False
    return True


def shape_extra_answer(old, new):
    try:
        if set(old) != set(new) or any(old[k] != new[k] for k in old if k != "answers"):
            return False
        o, n = old["answers"], new["answers"]
    except (TypeError, KeyError):
        return False
    if len(n) != len(o) + 1:
        return False
    for i in range(len(n)):
        if n[:i] + n[i + 1:] == o and (i > 0 and n[i] == n[i - 1] or i + 1 < len(n) and n[i] == n[i + 1]):
            return True
    return False


# The five known flakes of runs-audit.md section 3. kind "story": the failed spec files. kind "drift": the exact drifted file and the diff shape.
FLAKES = [
    {"name": "load-curve", "kind": "story", "file": LOAD_CURVE},
    {"name": "ctrl-c-silent", "kind": "story", "file": CTRL_C},
    {"name": "recorder-overview", "kind": "drift", "file": W + "standings/overview.read.json", "shape": shape_overview},
    {"name": "recorder-two-macs-up", "kind": "drift", "file": W + "two-macs/gestures/machines.create-workspace.json", "shape": shape_up},
    {"name": "recorder-two-macs-up", "kind": "drift", "file": W + "two-macs/gestures/machines.start.json", "shape": shape_up},
    {"name": "recorder-disk-read", "kind": "drift", "file": W + "mini-workspace/disk.read.json", "shape": shape_extra_answer},
]
RERUN_STEP = {"STORIES": "STORIES", "DRIFT": "GEN"}


def steps_of(body):
    failed, seen = [], set()
    m = re.search(r"^GOFMT\[(.*)\]", body, flags=re.M)
    if m:
        seen.add("GOFMT")
        if m.group(1).strip():
            failed.append("GOFMT")
    for k in ("LINT", "GEN", "GOTEST", "MAKELINT", "BUILD", "STORIES"):
        m = re.search(r"^" + k + r"=(\S+)", body, flags=re.M)
        if m:
            seen.add(k)
            if m.group(1) != "0":
                failed.append(k)
    m = re.search(r"^DRIFT\[(.*)\]", body, flags=re.M)
    drift = []
    if m:
        seen.add("DRIFT")
        drift = [x for x in re.split(r"\s+", m.group(1).strip()) if x and x not in ("M", "A", "D", "??", "R", "MM", "AM")]
        if drift:
            failed.append("DRIFT")
    if "GOFMT" in seen and ("VET-OK" not in body or "SPECSVET-OK" not in body):
        failed.append("VET")
    return failed, seen, drift


def stories_failed(body):
    files = set(re.findall(r"^ +❯ +\S+ +(\S+\.spec\.yaml) \(\d+ tests?", body, flags=re.M))
    if not files:
        files = set(re.findall(r"^ FAIL +\S+ +(\S+\.spec\.yaml) >", body, flags=re.M))
    return files


def drift_patch_files(patch):
    out, cur = {}, None
    for line in patch.split("\n"):
        m = re.match(r"diff --git a/(\S+) b/(\S+)", line)
        if m:
            cur = m.group(2)
            out[cur] = {"-": [], "+": [], "odd": False}
        elif cur and re.match(r"(new file|deleted file|rename |Binary|similarity)", line):
            out[cur]["odd"] = True
        elif cur and line.startswith("-") and not line.startswith("---"):
            out[cur]["-"].append(line[1:])
        elif cur and line.startswith("+") and not line.startswith("+++"):
            out[cur]["+"].append(line[1:])
    return out


def drift_name(files, patch, strict):
    byfile = {}
    for fl in FLAKES:
        if fl["kind"] == "drift":
            byfile.setdefault(fl["file"], []).append(fl)
    pf = drift_patch_files(patch) if patch else None
    names = []
    for f in files:
        key = next((k for k in byfile if f == k or f.endswith("/" + k)), None)
        if key is None:
            return None, "drift file " + f + " is no known flake's"
        sig = byfile[key][0]
        if pf is None:
            if strict:
                return None, "no drift patch to check the diff shape of " + f
        else:
            ent = pf.get(f)
            if ent is None or ent["odd"] or len(ent["-"]) != len(ent["+"]) or not ent["-"]:
                return None, "diff shape of " + f + " is not a one-line change per hunk"
            try:
                pairs = [(json.loads(o), json.loads(n)) for o, n in zip(ent["-"], ent["+"])]
            except ValueError:
                return None, "diff lines of " + f + " are not JSON"
            if not all(sig["shape"](o, n) for o, n in pairs):
                return None, "diff shape of " + f + " is not " + sig["name"]
        names.append(sig["name"])
    if set(pf or {}) - set(files):
        return None, "the patch touches more files than the drift line"
    return sorted(set(names)), ""


def judge(body, patch=None, only=None, strict=False):
    failed, seen, drift = steps_of(body)
    if only:
        want = {"GEN" if s in ("DRIFT", "GEN") else "STORIES" if s in ("1", "STORIES") else s for s in only.split(",")}
        failed = [s for s in failed if s in want or (s == "DRIFT" and "GEN" in want)]
        for s in want:
            if (s == "GEN" and "DRIFT" not in seen) or (s != "GEN" and s not in seen):
                failed.append("INCOMPLETE")
    elif "CC-DONE" not in body or not seen:
        failed.append("INCOMPLETE")
    rc = re.search(r"^REMOTE-RC=(\S+)", body, flags=re.M)
    if rc and rc.group(1) not in ("0",):
        failed.append("RC=" + rc.group(1))
    failed = sorted(set(failed))
    if not failed:
        return "GREEN", [], ""
    if set(failed) - {"STORIES", "DRIFT"}:
        return "RED", failed, ""
    names = []
    if "STORIES" in failed:
        files = stories_failed(body)
        known = {fl["file"]: fl["name"] for fl in FLAKES if fl["kind"] == "story"}
        if not files or not all(any(f == k or f.endswith("/" + k) for k in known) for f in files):
            return "RED", failed, "a failed story is no known flake's"
        names += [n for k, n in known.items() if any(f == k or f.endswith("/" + k) for f in files)]
    if "DRIFT" in failed:
        dn, why = drift_name(drift, patch, strict)
        if dn is None:
            return "RED", failed, why
        names += dn
    return "FLAKE(" + "+".join(sorted(set(names))) + ")", failed, ""


def blocks(path):
    txt = open(path, errors="replace").read()
    parts = re.split(r"^=== (\d+) ([0-9a-f]{40})\n", txt, flags=re.M)
    for i in range(1, len(parts), 3):
        yield int(parts[i]), parts[i + 1][:9], parts[i + 2]


def main(argv):
    if len(argv) >= 2 and argv[0] == "--block":
        patch = only = None
        a = argv[2:]
        while a:
            if a[0] == "--patch" and len(a) > 1:
                patch = open(a[1], errors="replace").read() if __import__("os").path.exists(a[1]) else None
                a = a[2:]
            elif a[0] == "--only" and len(a) > 1:
                only = a[1]
                a = a[2:]
            else:
                print(__doc__)
                return 2
        v, failed, why = judge(open(argv[1], errors="replace").read(), patch, only, strict=True)
        print(v, "steps=" + (",".join(failed) or "-"), why)
        return 1 if v == "RED" else 0
    if argv and argv[0] == "--replay":
        for path in argv[1:]:
            for n, sha, body in blocks(path):
                v, failed, why = judge(body)
                print(f"{path.rsplit('/', 1)[-1]} {n} {sha} {v} steps={','.join(failed) or '-'} {why}".rstrip())
        return 0
    print(__doc__)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
