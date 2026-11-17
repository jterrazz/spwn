"""value: the no-zoom measure. An opening is sampled frame by frame (the app
stretches its fade by STATION_TRIAL_SLOW), each frame a crop of the screen
around the panel, the first one taken before the click. It passes when:
the panel's outline is the same from its first visible frame to the last (within
BOX px), the header title, found in the last frame, sits at the same place in
every frame where it shows (within MOVE px) and as sharp as in the last
(SCORE), the title's contrast rises monotonically, the capture holds at least
FRAMES frames of the fade, and the app's own log shows one glass frame on every
display tick.
"""

import re

import numpy as np
from scipy import ndimage

BOX = 2
# value: a pixel is the panel's body when it differs from the closed screen by this much of 255,
# which leaves the shadow's halo out: the halo's reach grows 2 pt over the first 0.5 s.
CHANGED = 48
MOVE = 1
SCORE = 0.97
FRAMES = 10
RISE = 0.05
SHOWN = 0.3
BAR_PT = 30


def gray(rgb):
    return np.asarray(rgb, dtype=np.float64).mean(axis=2) / 255


def outline(frame, closed, bar):
    """The bounding box of what changed under the menu bar since the closed frame."""
    diff = np.abs(frame.astype(np.int16) - closed.astype(np.int16)).max(axis=2)
    diff = ndimage.median_filter(diff, size=3) > CHANGED
    diff[:bar] = False
    ys, xs = np.nonzero(diff)
    return None if len(ys) < 200 else (int(xs.min()), int(ys.min()), int(xs.max()) + 1, int(ys.max()) + 1)


def title(last, box):
    """The header title: the first text line of the last frame inside the outline,
    as a box of the frame."""
    x0, y0, x1, y1 = box
    g = gray(last)
    ink = np.abs(g - ndimage.median_filter(g, size=15)) > 0.12
    ink[:y0] = False
    ink[y0 + 200 :] = False
    ink[:, :x0] = False
    ink[:, x1:] = False
    labels, _ = ndimage.label(ndimage.binary_dilation(ink, structure=np.ones((3, 11))))
    best = None
    for sl in ndimage.find_objects(labels):
        h, w = sl[0].stop - sl[0].start, sl[1].stop - sl[1].start
        if 14 <= h <= 48 and w >= 2 * h and (best is None or sl[0].start < best[0].start):
            best = sl
    if best is None:
        return None
    return best[1].start, best[0].start, best[1].stop, best[0].stop


def match(frame, template, at, reach=12):
    """The offset of the template in the frame within reach px of where it is in
    the last frame, with its normalised correlation."""
    x0, y0, x1, y1 = at
    h, w = template.shape
    t = template - template.mean()
    best = (-2.0, 0, 0)
    g = gray(frame)
    for dy in range(-reach, reach + 1):
        for dx in range(-reach, reach + 1):
            if y0 + dy < 0 or x0 + dx < 0:
                continue
            c = g[y0 + dy : y0 + dy + h, x0 + dx : x0 + dx + w]
            if c.shape != template.shape:
                continue
            c = c - c.mean()
            d = np.sqrt((c * c).sum() * (t * t).sum())
            if d > 0 and (s := float((c * t).sum() / d)) > best[0]:
                best = (s, dx, dy)
    return best


def contrast(frame, at):
    x0, y0, x1, y1 = at
    g = gray(frame)[y0:y1, x0:x1]
    return float(np.percentile(g, 98) - np.percentile(g, 2))


def ticks(log):
    """The glass frames the app logged, one per display tick of the opening."""
    return [m.group(1) for m in re.finditer(r" open t=\d+ frame=\d+ glass=(.*)", log)]


def measure(frames, scale, log):
    """frames[0] is the closed screen, the others follow the click. Returns the
    numbers and the verdict."""
    closed, shots = frames[0], frames[1:]
    bar = int(BAR_PT * scale)
    last = shots[-1]
    final = outline(last, closed, bar)
    if final is None:
        return {"pass": False, "why": "the last frame shows no panel", "frames": len(shots)}
    boxes = [outline(f, closed, bar) for f in shots]
    visible = [i for i, b in enumerate(boxes) if b is not None]
    first = visible[0]
    off = max(max(abs(a - b) for a, b in zip(boxes[i], final)) for i in visible)
    at = title(last, final)
    if at is None:
        return {"pass": False, "why": "no title found in the last frame", "frames": len(shots)}
    template = gray(last)[at[1] : at[3], at[0] : at[2]]
    ref = contrast(last, at)
    moves, scores, alphas = [], [], []
    for f in shots[first:]:
        s, dx, dy = match(f, template, at)
        a = contrast(f, (at[0] + dx, at[1] + dy, at[2] + dx, at[3] + dy)) / ref
        alphas.append(round(min(a, 1.5), 3))
        if a >= SHOWN:
            moves.append(max(abs(dx), abs(dy)))
            scores.append(s)
    fade = sum(1 for a in alphas if 0.05 < a < 0.95)
    falls = max((alphas[i - 1] - alphas[i] for i in range(1, len(alphas))), default=0)
    seen = sorted(set(ticks(log)))
    out = {
        "frames": len(shots),
        "visible_from": first,
        "fade_frames": fade,
        "outline_px": off,
        "title_move_px": max(moves, default=-1),
        "title_score_min": round(min(scores, default=-1), 3),
        "alpha_fall": round(falls, 3),
        "alphas": alphas,
        "log_ticks": len(ticks(log)),
        "log_distinct_frames": len(seen),
    }
    checks = {
        "outline": off <= BOX,
        "title": 0 <= out["title_move_px"] <= MOVE and out["title_score_min"] >= SCORE,
        "rise": falls <= RISE,
        "frames": fade >= FRAMES,
        "ticks": out["log_ticks"] > 0 and len(seen) == 1,
    }
    out["checks"] = checks
    out["pass"] = all(checks.values())
    return out
