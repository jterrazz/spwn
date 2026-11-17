"""value: the measures' own test, run in a venv with pillow, numpy and scipy:
python3 -I measure_test.py.
run.sh --capture runs it before it boots anything.

The wash bars must hold the system's own pairs and refuse the build the baseline
records (baseline.json, the numbers of the capture at 47bedbecd); the zoom measure must
refuse a title that scales as the panel opens and accept one that does not.
"""

import json
import os
import sys
import unittest

import numpy as np
from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
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

    def test_the_menu_against_itself_passes_on_the_recorded_noise(self):
        base = json.load(open(os.path.join(HERE, "baseline.json")))["tint"]
        for row in base:
            self.assertLessEqual(row["noise"], tint.DELTA_E / 2, row["name"])


def scene(width, height):
    image = Image.new("RGB", (width, height), (120, 160, 200))
    return image


def frame(shift=1.0, alpha=1.0):
    """A closed-screen stand-in 400 x 600 with a panel 300 x 500 at (50, 80), its
    title drawn at `shift` of its size around the panel's top centre."""
    img = scene(400, 600)
    draw = ImageDraw.Draw(img)
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


def run(shifts, alphas):
    frames = [np.array(scene(400, 600))] + [frame(s, a) for s, a in zip(shifts, alphas)]
    return opening.measure(frames, 1.0, "")


class Zoom(unittest.TestCase):
    alphas = [0.2, 0.35, 0.5, 0.6, 0.7, 0.8, 0.85, 0.9, 0.95, 1.0, 1.0, 1.0]

    def test_a_title_that_scales_in_fails(self):
        shifts = [0.9, 0.92, 0.93, 0.94, 0.95, 0.96, 0.97, 0.98, 0.99, 1.0, 1.0, 1.0]
        out = run(shifts, self.alphas)
        self.assertFalse(out["pass"])
        self.assertFalse(out["checks"]["title"], out)

    def test_a_title_that_fades_in_place_passes(self):
        out = run([1.0] * 12, self.alphas)
        self.assertTrue(out["checks"]["title"], out)
        self.assertTrue(out["checks"]["outline"], out)
        self.assertTrue(out["checks"]["rise"], out)

    def test_a_fade_that_falls_back_fails(self):
        out = run([1.0] * 12, [0.2, 0.5, 0.9, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.0, 1.0])
        self.assertFalse(out["checks"]["rise"])

    def test_a_log_with_two_glass_frames_fails(self):
        log = "x open t=1 frame=1 glass=x=1 y=2 w=3 h=4\nx open t=2 frame=2 glass=x=1 y=2 w=3 h=5\n"
        frames = [np.array(scene(400, 600))] + [frame(1.0, a) for a in self.alphas]
        self.assertFalse(opening.measure(frames, 1.0, log)["checks"]["ticks"])

    def test_the_recorded_baseline_fails(self):
        base = json.load(open(os.path.join(HERE, "baseline.json")))["opening"]
        for row in base:
            self.assertFalse(row["pass"], row["name"])
            self.assertFalse(row["checks"]["title"], row["name"])


if __name__ == "__main__":
    unittest.main()
