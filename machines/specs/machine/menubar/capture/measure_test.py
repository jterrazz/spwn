"""value: the measures' own test, run in a venv with pillow, numpy and scipy:
python3 -I measure_test.py.
run.sh --capture runs it before it boots anything.

The wash bars must hold the system's own pairs and the owner's pick, and refuse the
builds the baseline records (baseline.json: the capture at 47bedbecd, and the tinted
glass of sweep mb1-sweep4); the zoom measure must refuse a title that scales or fades in
as the panel opens, and an item that drops its highlight, or a panel whose top is not
Passwords', and accept a panel shown whole under a lit item. The Dock's verdict must
refuse a bright white tile among grey glass in a mono appearance, and accept one inside
the Apple tiles' range. The bench's reader must time an opening to its first full frame,
and refuse a bench over an engine that never answered: every frame a skeleton, no watch logged.
"""

import json
import os
import sys
import unittest

import numpy as np
from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import dock  # noqa: E402
import bench  # noqa: E402
import opening  # noqa: E402
import tint  # noqa: E402


class Wash(unittest.TestCase):
    def test_ciede2000_matches_sharmas_reference_pair(self):
        self.assertAlmostEqual(tint.delta_e2000((50.0, 2.6772, -79.7751), (50.0, 0.0, -82.7485)), 2.0425, places=3)

    def test_the_system_against_the_system_passes(self):
        # the Tahoe capture: Passwords' panel and the Apple menu, one material, over a desk each
        for panel, menu in (((173, 228, 255), (177, 223, 253)), ((16, 68, 100), (17, 61, 94))):
            self.assertTrue(tint.verdict(tint.to_lab(np.array(panel)), tint.to_lab(np.array(menu)))["pass"])

    def test_the_recorded_baseline_fails_the_wash(self):
        base = json.load(open(os.path.join(HERE, "baseline.json")))["tint"]
        self.assertEqual(len(base), 6)
        failing = {row["name"] for row in base if not tint.verdict(np.array(row["panel_lab"]), np.array(row["menu_lab"]))["pass"]}
        # the pale desks hide a white tint and the panel's own colour moves between runs (dark-dark
        # read 4.23 in one and 1.33 in the next): the three desks that failed in both runs must fail
        self.assertTrue({"dark-color", "light-dark", "light-color"} <= failing, failing)

    def test_the_owners_pick_passes_on_every_desk_and_the_tint_it_replaced_fails(self):
        round6 = json.load(open(os.path.join(HERE, "baseline.json")))["round6"]
        verdicts = {k: {row["name"]: tint.verdict(np.array(row["panel_lab"]), np.array(row["menu_lab"]))["pass"] for row in round6[k]} for k in ("pick", "tint")}
        self.assertEqual(len(verdicts["pick"]), 6)
        self.assertTrue(all(verdicts["pick"].values()), verdicts["pick"])
        self.assertFalse(verdicts["tint"]["light-dark"], verdicts["tint"])

    def test_the_menu_against_itself_passes_on_the_recorded_noise(self):
        base = json.load(open(os.path.join(HERE, "baseline.json")))["tint"]
        for row in base:
            self.assertLessEqual(row["noise"], tint.DELTA_E / 2, row["name"])


def scene(width, height):
    image = Image.new("RGB", (width, height), (120, 160, 200))
    return image


ITEM = (300, 340)


def frame(shift=1.0, alpha=1.0, lit=True):
    """A closed-screen stand-in 400 x 600 with a panel 300 x 500 at (50, 80), its
    title drawn at `shift` of its size around the panel's top centre, and the status
    item's pill over ITEM's columns when lit."""
    img = scene(400, 600)
    draw = ImageDraw.Draw(img)
    if lit:
        draw.rounded_rectangle((ITEM[0], 4, ITEM[1], 26), 8, fill=(160, 190, 220))
    draw.rounded_rectangle((50, 80, 350, 580), 20, fill=(230, 235, 240))
    text = Image.new("RGB", (160, 36), (230, 235, 240))
    font = ImageFont.load_default(size=22)
    ImageDraw.Draw(text).text((4, 4), "All machines", fill=(0, 0, 0), font=font)
    if shift != 1.0:
        w, h = text.size
        scaled = text.resize((round(w * shift), round(h * shift)), Image.LANCZOS)
        text = Image.new("RGB", text.size, (230, 235, 240))
        text.paste(scaled, (round((w - scaled.width) / 2), 0))
    blank = Image.new("RGB", text.size, (230, 235, 240))
    img.paste(Image.blend(blank, text, alpha), (70, 100))
    return np.array(img)


def run(shifts, alphas, lits=None, log=""):
    lits = lits or [True] * len(shifts)
    frames = [np.array(scene(400, 600))] + [frame(s, a, lit) for s, a, lit in zip(shifts, alphas, lits)]
    return opening.measure(frames, 1.0, log, ITEM)


SCREEN = "x screen x=0.0 y=0.0 w=1024.0 h=768.0\n"
ONE_GLASS = SCREEN + "x open t=1 frame=1 glass=x=458.0 y=106.0 w=340.0 h=631.0\nx open t=2 frame=2 glass=x=458.0 y=106.0 w=340.0 h=631.0\n"


class Zoom(unittest.TestCase):
    alphas = [0.2, 0.35, 0.5, 0.6, 0.7, 0.8, 0.85, 0.9, 0.95, 1.0, 1.0, 1.0]

    def test_a_title_that_scales_in_fails(self):
        shifts = [0.9, 0.92, 0.93, 0.94, 0.95, 0.96, 0.97, 0.98, 0.99, 1.0, 1.0, 1.0]
        out = run(shifts, self.alphas)
        self.assertFalse(out["pass"])
        self.assertFalse(out["checks"]["title"], out)

    def test_a_title_that_fades_in_place_holds_still_but_is_not_whole(self):
        out = run([1.0] * 12, self.alphas)
        self.assertTrue(out["checks"]["title"], out)
        self.assertTrue(out["checks"]["outline"], out)
        self.assertTrue(out["checks"]["rise"], out)
        self.assertFalse(out["checks"]["whole"], out)
        self.assertFalse(out["pass"])

    def test_a_panel_shown_whole_under_a_lit_item_passes(self):
        out = run([1.0] * 12, [1.0] * 12, log=ONE_GLASS)
        self.assertTrue(out["pass"], out)
        self.assertEqual(out["title_move_px"], 0)
        self.assertGreaterEqual(out["title_score_min"], 0.99)

    def test_a_pill_that_lights_a_frame_after_the_panel_passes(self):
        out = run([1.0] * 12, [1.0] * 12, lits=[False] * 2 + [True] * 10, log=ONE_GLASS)
        self.assertTrue(out["checks"]["highlight"], out)
        self.assertEqual(out["pill_from"], 2)

    def test_an_item_never_lit_fails(self):
        out = run([1.0] * 12, [1.0] * 12, lits=[False] * 12, log=ONE_GLASS)
        self.assertFalse(out["checks"]["highlight"], out)

    def test_an_item_that_drops_its_highlight_fails(self):
        out = run([1.0] * 12, [1.0] * 12, lits=[True] * 3 + [False] * 9, log=ONE_GLASS)
        self.assertFalse(out["checks"]["highlight"], out)
        self.assertTrue(out["checks"]["whole"], out)

    def test_a_change_in_a_far_corner_is_not_the_panel(self):
        frames = [np.array(scene(400, 600))] + [frame(1.0, 1.0) for _ in range(12)]
        for f in frames[1:]:
            f[580:598, 380:398] = (250, 250, 250)
        out = opening.measure(frames, 1.0, ONE_GLASS, ITEM)
        self.assertTrue(out["checks"]["outline"], out)
        self.assertEqual(out["outline_px"], 0)

    def test_a_fade_that_falls_back_fails(self):
        out = run([1.0] * 12, [0.2, 0.5, 0.9, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.0, 1.0])
        self.assertFalse(out["checks"]["rise"])

    def test_a_log_with_two_glass_frames_fails(self):
        log = "x open t=1 frame=1 glass=x=1 y=2 w=3 h=4\nx open t=2 frame=2 glass=x=1 y=2 w=3 h=5\n"
        self.assertFalse(run([1.0] * 12, [1.0] * 12, log=log)["checks"]["ticks"])

    def test_a_panel_a_point_lower_than_passwords_fails(self):
        log = SCREEN + "x open t=1 frame=1 glass=x=458.0 y=106.0 w=340.0 h=630.0\n"
        out = run([1.0] * 12, [1.0] * 12, log=log)
        self.assertEqual(out["top_pt"], 32.0)
        self.assertFalse(out["checks"]["top"], out)

    def test_a_log_with_no_screen_line_fails_the_top(self):
        out = run([1.0] * 12, [1.0] * 12, log=ONE_GLASS.replace(SCREEN, ""))
        self.assertIsNone(out["top_pt"])
        self.assertFalse(out["checks"]["top"], out)

    def test_the_recorded_baseline_fails(self):
        base = json.load(open(os.path.join(HERE, "baseline.json")))["opening"]
        for row in base:
            self.assertFalse(row["pass"], row["name"])
            self.assertFalse(row["checks"]["title"], row["name"])


class Dock(unittest.TestCase):
    def capture(self, folder, theme, spwn):
        """A dock.json of one appearance: Apple's tiles grey from 60 to 110, Spwn Menu at
        90, and Spwn at `spwn`, each a flat 40 px tile."""
        greys = dict(zip(dock.APPLE, (60, 75, 90, 100, 110)))
        for name, grey in {**greys, "Spwn": spwn, "Spwn Menu": 90}.items():
            colour = grey if isinstance(grey, tuple) else (grey,) * 3
            Image.new("RGB", (40, 40), colour).save(dock.tile_path(folder, theme, name))
        Image.new("RGB", (300, 60), (50, 50, 50)).save(os.path.join(folder, f"dock-{theme}.png"))
        return {"theme": theme}

    def verdict(self, spwn):
        import tempfile

        with tempfile.TemporaryDirectory() as folder:
            docks = [self.capture(folder, theme, spwn) for theme in dock.MONO]
            json.dump(docks, open(os.path.join(folder, "dock.json"), "w"))
            ok = dock.verdict(folder)
            self.assertTrue(os.path.exists(os.path.join(folder, "board.png")))
            return ok

    def test_a_tile_inside_the_apple_range_passes(self):
        self.assertTrue(self.verdict(80))

    def test_a_white_tile_among_grey_glass_fails(self):
        self.assertFalse(self.verdict(250))

    def test_a_saturated_tile_among_grey_glass_fails(self):
        self.assertFalse(self.verdict((40, 140, 255)))


def bench_log(kind, frames):
    """An app's bench log of one opening per list of (t, height, generation)."""
    lines = []
    for i, opening_frames in enumerate(frames):
        lines.append(f"trial 1 bench {kind} {i}")
        lines += [f"trial 2 {kind} t={t} frame={n} glass=x=477.0 y=545.0 w=322.0 h={h} gen={g}"
                  for n, (t, h, g) in enumerate(opening_frames, 1)]
    return "\n".join(lines) + "\n"


WATCH = "2026-11-21T14:40:00Z watch: menubar first=12ms header=40ms full=95ms frames=3 commands=2\n"
DRAWN = [(20, 192.0, "skeleton-ddd6f82d"), (37, 192.0, "skeleton-ddd6f82d"), (54, 412.0, "1a2b3c4d"), (71, 630.0, "5e6f7a8b"),
         (88, 630.0, "5e6f7a8b")]
SKELETON = [(23, 192.0, "skeleton-ddd6f82d"), (39, 192.0, "skeleton-ddd6f82d"), (56, 192.0, "skeleton-ddd6f82d")]


class Bench(unittest.TestCase):
    def test_an_opening_is_timed_to_its_first_full_frame(self):
        self.assertEqual(bench.measure(DRAWN), {"full": 71, "first": 20, "gens": 3, "heights": 2, "skeleton": 2})

    def test_an_opening_left_on_its_skeleton_reaches_no_full_frame(self):
        self.assertIsNone(bench.measure(SKELETON)["full"])

    def test_a_bench_over_a_drawn_page_and_a_watch_measures(self):
        result = bench.bench(bench_log("open", [DRAWN]) + bench_log("push", [DRAWN]), WATCH)
        self.assertEqual(result["open"]["full"]["p50"], 71)
        self.assertEqual(result["watch menubar"]["full"]["p50"], 95)
        self.assertEqual(bench.refusals(result), [])

    def test_a_bench_over_an_engine_that_never_answered_fails(self):
        # the baseline run mbp-base2: the engine refused the macOS VM, the app drew its skeleton 50 times
        result = bench.bench(bench_log("open", [SKELETON] * 50) + bench_log("push", [SKELETON] * 50),
                             "Error: the engine runs only on a Mac; this machine is a macos VM\n")
        self.assertEqual(bench.refusals(result), ["no open opening reached a full frame", "no push opening reached a full frame",
                                                  "the engine logged no watch"])


if __name__ == "__main__":
    unittest.main()
