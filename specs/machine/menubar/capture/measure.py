"""value: the capture's contrast measure, M1's method on real pixels: each text
of the golden home.exposed panel is found and sorted by tone (t1, t2, the red
pill), the same box is taken in the capture, its pixels that are no glyph are
the glass behind the text, the token's colour is composited over each, and the
WCAG ratio's 10th percentile is the box's value. It prints a table, writes
measures.json and a compare image per shot: capture | M1 cell | golden.

usage: measure.py <out dir> <goldens dir> [M1 scenes dir]
"""

import json
import os
import sys

import numpy as np
from PIL import Image
from scipy import ndimage

# value: the tokens' two tones (Token.swift's colour table at 459cb381):
# t1 white or black, t2 white at 72 % or black at 62 %.
TONES = {"dark": {"t1": ((255, 255, 255), 1.0), "t2": ((255, 255, 255), 0.72)},
         "light": {"t1": ((0, 0, 0), 1.0), "t2": ((0, 0, 0), 0.62)}}
# value: a golden is the panel drawn with 24 pt of desk around it, at 2x.
PAD = 48
# value: Find's row at the panel's foot, 70 pt at 2x, pinned under the
# scroll when the panel stops at its cap.
FOOT = 140
TARGET = 4.5


def linear(c):
    c = c / 255.0
    return np.where(c <= 0.04045, c / 12.92, ((c + 0.055) / 1.055) ** 2.4)


def luminance(rgb):
    lin = linear(rgb.astype(np.float64))
    return 0.2126 * lin[..., 0] + 0.7152 * lin[..., 1] + 0.0722 * lin[..., 2]


def ratio(a, b):
    hi, lo = np.maximum(a, b), np.minimum(a, b)
    return (hi + 0.05) / (lo + 0.05)


def ink(rgb):
    gray = rgb.astype(np.float64).mean(axis=2) / 255
    return np.abs(gray - ndimage.median_filter(gray, size=15)) > 0.12


def shifted(golden, shift, foot):
    """The golden's glyph mask laid over a capture shorter by -shift: the
    head as it is, the foot moved up with Find."""
    mask = ink(golden)
    out = np.zeros((golden.shape[0] + shift, golden.shape[1]), dtype=bool)
    out[: out.shape[0] - foot] = mask[: out.shape[0] - foot]
    out[out.shape[0] - foot :] = mask[golden.shape[0] - foot :]
    return out


def boxes(golden, scheme):
    """Every text line of the golden, with its tone: a line is wider than twice
    its height, which leaves the symbols and the panel's corners out."""
    mask = ink(golden)
    labels, _ = ndimage.label(ndimage.binary_dilation(mask, structure=np.ones((3, 11))))
    out = []
    for sl in ndimage.find_objects(labels):
        h, w = sl[0].stop - sl[0].start, sl[1].stop - sl[1].start
        edge = sl[0].start < 8 or sl[1].start < 8 or sl[0].stop > golden.shape[0] - 8 or sl[1].stop > golden.shape[1] - 8
        if not (14 <= h <= 48 and w >= 2 * h) or edge:
            continue
        box = golden[sl].astype(np.float64)
        glyph = mask[sl]
        bg = np.median(box[~glyph], axis=0) if (~glyph).any() else np.zeros(3)
        if bg[0] > 150 and bg[1] < 90 and bg[2] < 90:
            if w >= 3 * h:
                out.append(("red", sl))
            continue
        px = box[glyph]
        if len(px) < 20:
            continue
        gray = px.mean(axis=1)
        core = px[gray >= np.percentile(gray, 90)] if scheme == "dark" else px[gray <= np.percentile(gray, 10)]
        tone = core.mean(axis=0)
        if tone.max() - tone.min() > 60:
            continue
        g = tone.mean() / 255
        if scheme == "dark":
            kind = "t1" if g >= 0.9 else "t2" if g >= 0.55 else None
        else:
            kind = "t1" if g <= 0.15 else "t2" if g <= 0.55 else None
        if kind:
            out.append((kind, sl))
    return out


def measure(capture, glyphs, kind, sl, scheme):
    box = capture[sl].astype(np.float64)
    glyph = ndimage.binary_dilation(glyphs[sl], iterations=2)
    bg = box[~glyph]
    if len(bg) < 20:
        return None
    if kind == "red":
        color, alpha = (255, 255, 255), 1.0
    else:
        color, alpha = TONES[scheme][kind]
    text = np.array(color, dtype=np.float64) * alpha + bg * (1 - alpha)
    r = ratio(luminance(text), luminance(bg))
    return float(np.percentile(r, 10))


def compare(name, capture, m1, golden, out):
    h = capture.shape[0]
    tiles = [Image.fromarray(capture)]
    for other in (m1, golden):
        if other is not None:
            img = Image.fromarray(other)
            tiles.append(img.resize((int(img.width * h / img.height), h)))
    sheet = Image.new("RGB", (sum(t.width for t in tiles) + 24 * (len(tiles) - 1), h), (40, 40, 44))
    x = 0
    for t in tiles:
        sheet.paste(t, (x, 0))
        x += t.width + 24
    sheet.save(os.path.join(out, f"{name}.compare.png"))


def main():
    out, goldens = sys.argv[1], sys.argv[2]
    scenes = sys.argv[3] if len(sys.argv) > 3 else None
    shots = json.load(open(os.path.join(out, "shots.json")))
    found = {}
    rows = []
    for shot in shots:
        name, scheme = shot["name"], shot["scheme"]
        golden_full = np.array(Image.open(os.path.join(goldens, f"home.exposed.{scheme}.png")).convert("RGB"))
        golden = golden_full[PAD:-PAD, PAD:-PAD]
        if scheme not in found:
            found[scheme] = boxes(golden, scheme)
        capture = np.array(Image.open(os.path.join(out, f"{name}.png")).convert("RGB"))
        h, w = min(capture.shape[0], golden.shape[0]), min(capture.shape[1], golden.shape[1])
        shift = min(capture.shape[0] - golden.shape[0], 0)
        glyphs = (shifted(golden, shift, FOOT) if shift else ink(golden))[: capture.shape[0], : capture.shape[1]]
        glyphs = glyphs | ink(capture)[: glyphs.shape[0], : glyphs.shape[1]]
        worst = {}
        for kind, sl in found[scheme]:
            # invariant: a panel at its cap scrolls between a fixed head and
            # Find, pinned to its bottom: a text in the golden's last
            # FOOT pixels is read that far from the capture's bottom, and a
            # text the scroll hid is not read.
            if sl[0].start >= golden.shape[0] - FOOT:
                sl = (slice(sl[0].start + shift, sl[0].stop + shift), sl[1])
            elif sl[0].stop > h - FOOT:
                continue
            if sl[1].stop > w:
                continue
            v = measure(capture, glyphs, kind, sl, scheme)
            if v is None:
                continue
            at = f"y={sl[0].start // 2}pt x={sl[1].start // 2}pt"
            if kind not in worst or v < worst[kind][0]:
                worst[kind] = (v, at)
        m1 = None
        if scenes:
            path = os.path.join(scenes, f"M1-{scheme}_{shot['wall']}.png")
            if os.path.exists(path):
                m1 = np.array(Image.open(path).convert("RGB"))[68:810, 80:760]
        compare(name, capture, m1, golden, out)
        row = {"name": name, "size": [int(capture.shape[1]), int(capture.shape[0])], "golden": [int(golden.shape[1]), int(golden.shape[0])],
               **{k: {"p10": round(v[0], 2), "at": v[1]} for k, v in worst.items()}}
        row["pass"] = all(row.get(k, {"p10": 0})["p10"] >= TARGET for k in ("t2", "red"))
        rows.append(row)
    json.dump({"boxes": {k: len(v) for k, v in found.items()}, "rows": rows}, open(os.path.join(out, "measures.json"), "w"), indent=1)
    counts = [f"{k} {len(v)} ({sum(1 for b in v if b[0] == 't2')} t2)" for k, v in found.items()]
    print(f"text boxes found in the goldens: {', '.join(counts)}")
    print("| shot | t1 | t2 (worst, where) | red pill | 4.5:1 |")
    print("|---|---|---|---|---|")
    for r in rows:
        cell = lambda k: f"{r[k]['p10']:.2f} ({r[k]['at']})" if k in r else "n/a"
        print(f"| {r['name']} | {r['t1']['p10'] if 't1' in r else 'n/a'} | {cell('t2')} | {r['red']['p10'] if 'red' in r else 'n/a'} | {'pass' if r['pass'] else 'FAIL'} |")


if __name__ == "__main__":
    main()
