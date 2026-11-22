"""value: the Dock's verdict, from the dock set's captures (dock.json): in each
of the four mono appearances (clear and tinted, light and dark), every judged
tile's mean luminance and saturation, read over the middle of the tile, fall
inside the range of the Apple tiles of the same capture. A Spwn tile left
bright white among grey glass, as the flat icon was, falls outside it. The
regular appearances are drawn and measured, and judged by eye.

It writes dock-<theme>-<app>-diff.png (the judged tile against the Apple tile
nearest to it, the difference times DIFF_GAIN), dock-verdict.json, and board.png:
a row per appearance, the real Dock beside its numbers and diffs.
"""

import json
import os

import numpy as np
from PIL import Image, ImageDraw, ImageFont

APPLE = ["Finder", "Safari", "Notes", "System Settings", "Terminal"]
JUDGED = ["Spwn"]
MONO = ["ClearLight", "ClearDark", "TintedLight", "TintedDark"]
# value: the middle of a tile, clear of its rounded corners and of the Dock's glass.
INNER = 0.6
DIFF_GAIN = 3
ROW_H = 180


def tile_path(out, theme, name):
    return os.path.join(out, f"dock-{theme}-{name.replace(' ', '')}.png")


def stats(path):
    rgb = np.asarray(Image.open(path).convert("RGB"), dtype=np.float64) / 255.0
    h, w = rgb.shape[:2]
    m0, m1 = int(h * (1 - INNER) / 2), int(w * (1 - INNER) / 2)
    mid = rgb[m0 : h - m0, m1 : w - m1]
    lin = np.where(mid <= 0.04045, mid / 12.92, ((mid + 0.055) / 1.055) ** 2.4)
    lum = 0.2126 * lin[..., 0] + 0.7152 * lin[..., 1] + 0.0722 * lin[..., 2]
    hi, lo = mid.max(axis=2), mid.min(axis=2)
    sat = np.where(hi > 0, (hi - lo) / np.maximum(hi, 1e-9), 0)
    return {"lum": round(float(lum.mean()), 4), "sat": round(float(sat.mean()), 4)}


def diff(out, theme, judged, nearest):
    a = np.asarray(Image.open(tile_path(out, theme, judged)).convert("RGB"), dtype=np.int16)
    b = np.asarray(Image.open(tile_path(out, theme, nearest)).convert("RGB").resize((a.shape[1], a.shape[0])), dtype=np.int16)
    d = np.clip(np.abs(a - b) * DIFF_GAIN, 0, 255).astype(np.uint8)
    path = os.path.join(out, f"dock-{theme}-{judged.replace(' ', '')}-diff.png")
    Image.fromarray(d).save(path)
    return path


def judge(out):
    docks = json.load(open(os.path.join(out, "dock.json")))
    rows = []
    for d in docks:
        theme = d["theme"]
        numbers = {n: stats(tile_path(out, theme, n)) for n in APPLE + JUDGED}
        lo = {k: min(numbers[n][k] for n in APPLE) for k in ("lum", "sat")}
        hi = {k: max(numbers[n][k] for n in APPLE) for k in ("lum", "sat")}
        judged = {}
        for n in JUDGED:
            v = numbers[n]
            nearest = min(APPLE, key=lambda a: abs(numbers[a]["lum"] - v["lum"]) + abs(numbers[a]["sat"] - v["sat"]))
            inside = all(lo[k] <= v[k] <= hi[k] for k in ("lum", "sat"))
            judged[n] = {**v, "nearest": nearest, "diff": os.path.basename(diff(out, theme, n, nearest)),
                         "inside": inside, "pass": inside if theme in MONO else None}
        rows.append({"theme": theme, "apple": {"lo": lo, "hi": hi}, "numbers": numbers, "judged": judged})
    return rows


def board(out, rows):
    font = ImageFont.load_default(size=22)
    small = ImageFont.load_default(size=17)
    docks = [Image.open(os.path.join(out, f"dock-{r['theme']}.png")).convert("RGB") for r in rows]
    scaled = [im.resize((int(im.width * ROW_H / im.height), ROW_H)) for im in docks]
    dock_w = max(im.width for im in scaled)
    cell = ROW_H
    width = 200 + dock_w + 40 + len(JUDGED) * (3 * cell + 320) + 20
    height = 90 + len(rows) * (ROW_H + 24) + 20
    sheet = Image.new("RGB", (width, height), (246, 246, 246))
    draw = ImageDraw.Draw(sheet)
    draw.text((20, 18), "The Dock, captured in the menu bar rig's VM, in each icon appearance", fill=(0, 0, 0), font=font)
    draw.text((20, 50), f"judged tile | nearest Apple tile | difference x{DIFF_GAIN}; inside = luminance and saturation within the Apple tiles' range", fill=(80, 80, 80), font=small)
    for i, (r, im) in enumerate(zip(rows, scaled)):
        y = 90 + i * (ROW_H + 24)
        verdicts = [j["pass"] for j in r["judged"].values()]
        mark = "judged by eye" if r["theme"] not in MONO else ("PASS" if all(verdicts) else "FAIL")
        draw.text((20, y + 50), r["theme"], fill=(0, 0, 0), font=font)
        draw.text((20, y + 84), mark, fill=(0, 120, 0) if mark == "PASS" else (180, 0, 0) if mark == "FAIL" else (80, 80, 80), font=font)
        sheet.paste(im, (200, y))
        x = 200 + dock_w + 40
        lo, hi = r["apple"]["lo"], r["apple"]["hi"]
        for n, j in r["judged"].items():
            for k, path in enumerate((tile_path(out, r["theme"], n), tile_path(out, r["theme"], j["nearest"]), os.path.join(out, j["diff"]))):
                sheet.paste(Image.open(path).convert("RGB").resize((cell, cell)), (x + k * cell, y))
            tx = x + 3 * cell + 12
            draw.text((tx, y + 8), f"{n} vs {j['nearest']}", fill=(0, 0, 0), font=small)
            draw.text((tx, y + 36), f"lum {j['lum']:.3f}  [{lo['lum']:.3f}, {hi['lum']:.3f}]", fill=(0, 0, 0), font=small)
            draw.text((tx, y + 62), f"sat {j['sat']:.3f}  [{lo['sat']:.3f}, {hi['sat']:.3f}]", fill=(0, 0, 0), font=small)
            draw.text((tx, y + 92), "inside" if j["inside"] else "outside", fill=(0, 120, 0) if j["inside"] else (180, 0, 0), font=font)
            x += 3 * cell + 320
    path = os.path.join(out, "board.png")
    sheet.save(path)
    return path


def verdict(out):
    rows = judge(out)
    path = board(out, rows)
    json.dump(rows, open(os.path.join(out, "dock-verdict.json"), "w"), indent=1)
    print("| appearance | tile | lum | Apple lum | sat | Apple sat | nearest | verdict |")
    print("|---|---|---|---|---|---|---|---|")
    ok = True
    for r in rows:
        lo, hi = r["apple"]["lo"], r["apple"]["hi"]
        for n, j in r["judged"].items():
            word = "eye" if j["pass"] is None else "PASS" if j["pass"] else "FAIL"
            ok = ok and j["pass"] is not False
            print(f"| {r['theme']} | {n} | {j['lum']:.3f} | {lo['lum']:.3f}-{hi['lum']:.3f} | {j['sat']:.3f} | {lo['sat']:.3f}-{hi['sat']:.3f} | {j['nearest']} ({j['diff']}) | {word} |")
    missing = sorted(set(MONO) - {r["theme"] for r in rows})
    if missing:
        print(f"dock: no capture of {', '.join(missing)}")
        ok = False
    print(f"dock: {'PASS' if ok else 'FAIL'}, board {path}")
    return ok
