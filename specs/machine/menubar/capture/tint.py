"""value: the wash measure. The panel's colour over a desk is its glyph-free
background's median in CIELAB over a rectangle both surfaces cover, and the
system menu's colour over the same desk is read the same way; the panel passes
when it is within DELTA_E of the menu (CIEDE2000) and within DELTA_B on b*,
the axis the owner's "white-bluish" is on, on every wall in both schemes.

The bars are calibrated on the system itself: Passwords' own panel against the
Apple menu, which the Tahoe capture (refs/passwords-tahoe) measured as one
material, is under both, and so is one Apple menu opened twice.
"""

import math

import numpy as np
from scipy import ndimage

DELTA_E = 4.0
DELTA_B = 2.0
PAD_PT = 12


def to_lab(rgb):
    c = np.asarray(rgb, dtype=np.float64) / 255.0
    c = np.where(c <= 0.04045, c / 12.92, ((c + 0.055) / 1.055) ** 2.4)
    m = np.array([[0.4124564, 0.3575761, 0.1804375], [0.2126729, 0.7151522, 0.0721750], [0.0193339, 0.1191920, 0.9503041]])
    xyz = c @ m.T / np.array([0.95047, 1.0, 1.08883])
    f = np.where(xyz > 216 / 24389, np.cbrt(xyz), (24389 / 27 * xyz + 16) / 116)
    return np.stack([116 * f[..., 1] - 16, 500 * (f[..., 0] - f[..., 1]), 200 * (f[..., 1] - f[..., 2])], axis=-1)


def delta_e2000(a, b):
    l1, a1, b1 = a
    l2, a2, b2 = b
    c1, c2 = math.hypot(a1, b1), math.hypot(a2, b2)
    g = 0.5 * (1 - math.sqrt(((c1 + c2) / 2) ** 7 / (((c1 + c2) / 2) ** 7 + 25**7)))
    a1p, a2p = (1 + g) * a1, (1 + g) * a2
    c1p, c2p = math.hypot(a1p, b1), math.hypot(a2p, b2)
    h1p = math.degrees(math.atan2(b1, a1p)) % 360
    h2p = math.degrees(math.atan2(b2, a2p)) % 360
    dl, dc = l2 - l1, c2p - c1p
    dh = 0.0
    if c1p * c2p:
        d = h2p - h1p
        dh = d - 360 if d > 180 else d + 360 if d < -180 else d
    dhp = 2 * math.sqrt(c1p * c2p) * math.sin(math.radians(dh / 2))
    lp, cp = (l1 + l2) / 2, (c1p + c2p) / 2
    hp = h1p + h2p
    if c1p * c2p:
        hp = hp / 2 if abs(h1p - h2p) <= 180 else (hp + 360) / 2 if hp < 360 else (hp - 360) / 2
    t = 1 - 0.17 * math.cos(math.radians(hp - 30)) + 0.24 * math.cos(math.radians(2 * hp)) + 0.32 * math.cos(math.radians(3 * hp + 6)) - 0.20 * math.cos(math.radians(4 * hp - 63))
    sl = 1 + 0.015 * (lp - 50) ** 2 / math.sqrt(20 + (lp - 50) ** 2)
    sc, sh = 1 + 0.045 * cp, 1 + 0.015 * cp * t
    rt = -2 * math.sqrt(cp**7 / (cp**7 + 25**7)) * math.sin(math.radians(60 * math.exp(-(((hp - 275) / 25) ** 2))))
    return math.sqrt((dl / sl) ** 2 + (dc / sc) ** 2 + (dhp / sh) ** 2 + rt * (dc / sc) * (dhp / sh))


def region(rects, scale):
    """The rectangle, in each surface's own pixels from its top-left, that every
    surface covers, less PAD_PT on each side."""
    pad = int(PAD_PT * scale)
    w = min(r[2] - r[0] for r in rects) - 2 * pad
    h = min(r[3] - r[1] for r in rects) - 2 * pad
    return pad, w, h


def median_lab(image, rect, box):
    """The median CIELAB of the glyph-free pixels of a surface's box."""
    pad, w, h = box
    x0, y0 = rect[0] + pad, rect[1] + pad
    px = np.asarray(image, dtype=np.float64)[y0 : y0 + h, x0 : x0 + w]
    gray = px.mean(axis=2) / 255
    ink = ndimage.binary_dilation(np.abs(gray - ndimage.median_filter(gray, size=15)) > 0.12, iterations=2)
    return np.median(to_lab(px[~ink]), axis=0), float(ink.mean())


def verdict(panel, menu):
    de = delta_e2000(panel, menu)
    db = float(panel[2] - menu[2])
    return {"dE": round(de, 2), "db": round(db, 2), "pass": de <= DELTA_E and abs(db) <= DELTA_B}
