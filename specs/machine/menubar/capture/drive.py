"""value: the capture's driver, run on the Mac by run.sh --capture: it plays the
built Station.app in the guest over the fake engine, opens its panel by a
click over the Mac's VNC framebuffer, and reads the panel back as pixels. No
window opens on the Mac, and the guest needs no Screen Recording grant.

usage: drive.py <ip> <ssh key> <vnc url> <repo> <out dir> <set> [old item x,y,w,h]
  set is a comma list of: walls (3 walls x dark/light), a11y (Reduce
  Transparency, Increase Contrast), trial (ADR-021's height and focus trial), behave (closing gestures), old (a
  build with no frame log: the item's frame is given and the panel is found
  by what the click changed), native (the Apple menu over the same walls, twice,
  for the wash measure) and opening (the panel's opening, sampled frame by
  frame after a click held as a hand holds it).
"""

import json
import os
import re
import subprocess
import sys
import time

from vncdotool import api

WALLS = ["dark", "light", "color"]
SCHEMES = ["dark", "light"]
APP = "~/station/.artifacts/menubar/Station.app"
CAPTURE = "~/station/specs/machine/menubar/capture"
# value: STATION_CAPTURE_PAGE names the home story served, home.exposed by
# default; home.empty is shorter than the cap on any screen.
STORY = os.environ.get("STATION_CAPTURE_PAGE", "home.exposed")
PAGE = f"~/station/apps/menubar/stories/pages/machines.{STORY}.menubar.json"
# value: the panel's top-left in M1's scene, from the item, as StatusPanel
# places it: its right edge under the item's, 6 pt under the menu bar.
DROP, MARGIN, WIDTH = 6, 8, 322
# value: the Apple menu opens at the bar's left, 30.5 pt under the top (the Tahoe
# capture, refs/passwords-tahoe rows 6 and 9); the native shots lay the desk's
# scene under it as under the panel.
MENU_AT = (5.0, 30.5)
APPLE = (20, 12)
SPAN = 5.0
# value: a hand's click holds the button about a tenth of a second, through which the
# status button tracks the mouse and may drop its highlight when it lets go.
HOLD = 0.15


class Rig:
    def __init__(self, ip, key, url, repo, out):
        self.ip, self.key, self.repo, self.out = ip, key, repo, out
        m = re.match(r"vnc://:([^@]*)@([\d.]+):(\d+)", url)
        self.server, self.password = f"{m.group(2)}::{m.group(3)}", m.group(1)
        self.vnc = api.connect(self.server, password=self.password)
        self.vnc.refreshScreen()
        self.fb = self.vnc.screen.size
        print(f"   framebuffer {self.fb[0]}x{self.fb[1]}", flush=True)
        self.wall = None
        self.walls = 0
        self.noitem = 0
        self.screen = None
        self.log = open(os.path.join(out, "drive.log"), "a", encoding="utf-8")

    def say(self, line):
        print(line, flush=True)
        self.log.write(line + "\n")
        self.log.flush()

    def ssh(self, cmd, check=False):
        # hazard: the Mac's own ssh config and agent are not read: an agent
        # holding many keys spends the guest's auth tries before this run's
        # key, and every call came back empty (runs cap5, cap6, fix1).
        argv = ["ssh", "-n", "-F", "/dev/null", "-o", "IdentitiesOnly=yes", "-o", "IdentityAgent=none",
                "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR",
                "-o", "BatchMode=yes", "-i", self.key, f"admin@{self.ip}", cmd]
        r = subprocess.run(argv, capture_output=True, text=True, timeout=300)
        if r.returncode == 255:
            raise RuntimeError(f"ssh admin@{self.ip}: {r.stderr.strip()}")
        if check and r.returncode != 0:
            raise RuntimeError(f"{cmd}: {r.returncode} {r.stderr.strip()}")
        return r.stdout

    # hazard: launchctl asuser keeps sudo's root; the second sudo runs the
    # command as admin, in admin's GUI session, so a default it writes is
    # admin's and the app runs as the owner would.
    def gui(self, cmd, check=False):
        return self.ssh(f"sudo -n launchctl asuser $(id -u) sudo -n -u admin {cmd}", check)

    def setup(self):
        revision = re.search(r"pageRevision = (\d+)", open(os.path.join(self.repo, "apps/menubar/sources/station-kit/page/Page.swift"), encoding="utf-8").read()).group(1)
        self.ssh("mkdir -p ~/cap ~/.local/state/station; pkill -f '[r]eplay.py'; true")
        self.ssh("printf 'tell application \"Terminal\" to quit\\n' > ~/cap/quit.scpt")
        self.gui("osascript ~/cap/quit.scpt")
        self.ssh(f"nohup python3 {CAPTURE}/replay.py {PAGE} ~/.local/state/station/engine.sock {revision} > ~/cap/replay.log 2>&1 < /dev/null &")
        self.wake()
        self.settle()

    # hazard: the framebuffer is the login screen's 1280 x 720 until the guest's session
    # sets its display (2048 x 1536 here), and a size read at connect time mapped every
    # click to the wrong pixel (run mb0-cap, 2026-11-17). A connection opened before
    # that change keeps asking for the old 1280 x 720 alone, so the rest of every shot
    # stays the frame it first read (run mb1-cap, 2026-11-17): it is opened again.
    def settle(self):
        for _ in range(45):
            self.vnc.refreshScreen()
            if self.vnc.screen.size != (1280, 720):
                break
            self.wake()
            time.sleep(2)
        if self.vnc.screen.size != self.fb:
            self.vnc.disconnect()
            self.vnc = api.connect(self.server, password=self.password)
            self.vnc.refreshScreen()
        self.fb = self.vnc.screen.size
        self.say(f"   framebuffer now {self.fb[0]}x{self.fb[1]}")

    # hazard: the server sends a client only the rectangles that changed, whatever the
    # refresh asks, so the copy of a screen that has not moved since the display was set
    # keeps the login's wallpaper and no status item (runs mb1b-cap, mb1b-cap2); a new
    # connection starts from the whole screen.
    def reconnect(self):
        self.vnc.disconnect()
        self.vnc = api.connect(self.server, password=self.password)
        self.vnc.refreshScreen()
        self.say(f"   reconnected, framebuffer {self.vnc.screen.size[0]}x{self.vnc.screen.size[1]}")

    def wake(self):
        self.vnc.mouseMove(self.fb[0] // 2, self.fb[1] // 2)
        self.vnc.mouseMove(self.fb[0] // 2 + 4, self.fb[1] // 2 + 4)
        time.sleep(0.3)

    def appearance(self, scheme, reduce=False, contrast=False, keyboard=None):
        dark = "true" if scheme == "dark" else "false"
        self.ssh(f"printf 'tell application \"System Events\" to tell appearance preferences to set dark mode to {dark}\\n' > ~/cap/mode.scpt")
        self.gui("osascript ~/cap/mode.scpt")
        for k, on in (("reduceTransparency", reduce), ("increaseContrast", contrast)):
            self.gui(f"defaults write com.apple.universalaccess {k} -bool {'true' if on else 'false'}")
        if keyboard is not None:
            self.gui(f"defaults write -g AppleKeyboardUIMode -int {keyboard}")
        read = self.gui("defaults read com.apple.universalaccess reduceTransparency; defaults read com.apple.universalaccess increaseContrast").split()
        return read

    def launch(self, env):
        self.ssh("pkill -x StationMenuBar; sleep 1; rm -f ~/cap/app.log; true")
        flags = " ".join(f"--env {k}={v}" for k, v in env.items())
        self.gui(f"open -n {APP} {flags} --stderr /Users/admin/cap/app.log --stdout /Users/admin/cap/app.out")
        if not any(k.startswith("STATION_TRIAL") for k in env):
            time.sleep(5)
            return None
        # hazard: the app logs its item before the bar has placed it (a y past the
        # screen's top) and its screen a moment later; a run that read the first
        # item with no screen line clicked outside the framebuffer (run mb0-cap, 2026-11-17).
        for _ in range(90):
            time.sleep(1)
            line, screen = self.last("item "), self.last("screen ")
            if line and screen:
                self.screen = self.frame(screen)
                if self.frame(line)["y"] + self.frame(line)["h"] <= self.screen["h"]:
                    self.say(f"   {line.split(' ', 2)[-1]}; {screen.split(' ', 2)[-1]}")
                    return line
        self.say(f"   no item frame in 90 s; app: {self.ssh('pgrep -lx StationMenuBar; tail -5 ~/cap/app.log ~/cap/app.out 2>&1').strip()}")
        self.noitem += 1
        self.capture(f"noitem-{self.noitem}")
        menu = self.gui("defaults read -g _HIHideMenuBar; defaults read com.apple.controlcenter AutoHideMenuBarOption; defaults -currentHost read com.apple.controlcenter")
        self.say(f"   noitem-{self.noitem}.full.png; menu bar settings: {' '.join(menu.split())[:400]}")
        return None

    def last(self, word):
        lines = [l for l in self.ssh("cat ~/cap/app.log 2>/dev/null").splitlines() if f" {word}" in l]
        return lines[-1] if lines else None

    @staticmethod
    def frame(line):
        return {k: float(v) for k, v in re.findall(r"\b([xywh])=(-?[\d.]+)", line)}

    # invariant: points map to the framebuffer by the screen's frame the
    # app logs; a build that logs none is read at 2x.
    def scale(self, item):
        height = self.screen["h"] if self.screen else self.fb[1] / 2
        return self.fb[1] / height, height

    def set_wall(self, kind, item):
        s, height = self.scale(item)
        width = self.fb[0] / s
        left = min(max(item["x"] + item["w"] - WIDTH, MARGIN), width - MARGIN - WIDTH)
        self.place(kind, left, height - (item["y"] - DROP))

    def place(self, kind, left, top):
        if self.wall == (kind, left, top):
            return
        self.walls += 1
        out = self.gui(f"swift {CAPTURE}/wall.swift {kind} /Users/admin/cap/wall-{self.walls}.png {left} {top}")
        self.say(f"   {out.strip()}")
        self.wall = (kind, left, top)
        time.sleep(2)

    def px(self, item, x, y):
        s, height = self.scale(item)
        return int(x * s), int((height - y) * s)

    def click_item(self, item, hold=0.0):
        self.wake()
        x, y = self.px(item, item["x"] + item["w"] / 2, item["y"] + item["h"] / 2)
        self.vnc.mouseMove(x, y)
        if not hold:
            self.vnc.mousePress(1)
            return
        self.vnc.mouseDown(1)
        time.sleep(hold)
        self.vnc.mouseUp(1)

    def park(self):
        self.vnc.mouseMove(4, self.fb[1] - 4)

    def capture(self, name):
        path = os.path.join(self.out, f"{name}.full.png")
        self.vnc.refreshScreen()
        self.vnc.captureScreen(path)
        return path

    def crop(self, name, full, item, window, margin=56):
        from PIL import Image

        s, height = self.scale(item)
        img = Image.open(full).convert("RGB")
        x, y = int(round(window["x"] * s)), int(round((height - window["y"] - window["h"]) * s))
        w, h = int(round(window["w"] * s)), int(round(window["h"] * s))
        img.crop((x, y, x + w, y + h)).save(os.path.join(self.out, f"{name}.png"))
        m = int(margin * s)
        img.crop((x - m, y - m, x + w + m, y + h + m)).save(os.path.join(self.out, f"{name}.wide.png"))

    def shot(self, name, scheme, wall, env, reduce=False, contrast=False, old_item=None):
        read = self.appearance(scheme, reduce, contrast)
        line = self.launch(env)
        item = old_item or (self.frame(line) if line else None)
        if not item:
            self.say(f"FAIL {name}: the app logged no item frame")
            return None
        self.set_wall(wall, item)
        self.park()
        time.sleep(1)
        before = self.capture(f"{name}.closed")
        self.click_item(item)
        time.sleep(2.5)
        self.park()
        time.sleep(1)
        full = self.capture(name)
        if old_item:
            window = self.changed(before, full, item)
        else:
            opened, window_line, content = self.last("opened"), self.last("glass "), self.last("content ")
            if not window_line:
                self.say(f"FAIL {name}: no glass frame logged")
                return None
            window = self.frame(window_line.split(" window ")[0].split(" glass ")[-1])
            edge = self.edge(before, full, item, self.frame(window_line.split(" window ")[1]))
            self.say(f"   {content.split(' ', 1)[-1] if content else 'no content size'}; the window's edge moved by at most {edge}")
            self.say(f"   {opened.split(' ', 2)[-1] if opened else 'not opened'}; {window_line.split(' ', 2)[-1]}")
        self.crop(name, full, item, window)
        self.vnc.keyPress("esc")
        time.sleep(1)
        closed = None if old_item else bool(self.last("closed"))
        s, height = self.scale(item)
        rect = None if old_item else [int(round(window["x"] * s)), int(round((height - window["y"] - window["h"]) * s)),
                                      int(round((window["x"] + window["w"]) * s)), int(round((height - window["y"]) * s))]
        meta = {"name": name, "story": STORY, "rect_px": rect, "scheme": scheme, "wall": wall, "env": env, "reduce": reduce, "contrast": contrast,
                "a11y_read": read, "item": item, "window": window, "esc_closed": closed}
        if not old_item:
            meta["edge"] = edge
        self.say(f"-- {name}: {json.dumps(meta)}")
        return meta

    # invariant: the window's own edge is the wallpaper: its outermost
    # 2 px ring reads the same open as closed, so no square frame shows
    # past the panel's shadow.
    def edge(self, before, after, item, frame):
        from PIL import Image, ImageChops

        s, height = self.scale(item)
        diff = ImageChops.difference(Image.open(before).convert("L"), Image.open(after).convert("L"))
        x0, y0 = int(round(frame["x"] * s)), int(round((height - frame["y"] - frame["h"]) * s))
        x1, y1 = x0 + int(round(frame["w"] * s)), y0 + int(round(frame["h"] * s))
        # value: the menu bar, which the open item highlights, is no edge.
        bar = int(30 * s)
        top = max(y0, bar)
        x0, x1, y1 = max(x0, 0), min(x1, diff.width), min(y1, diff.height)
        bands = [(x0, y1 - 2, x1, y1), (x0, top, x0 + 2, y1), (x1 - 2, top, x1, y1)] + ([(x0, y0, x1, y0 + 2)] if y0 >= bar else [])
        return max(diff.crop(b).getextrema()[1] for b in bands)

    def changed(self, before, after, item):
        from PIL import Image, ImageChops

        s, height = self.scale(item)
        a, b = Image.open(before).convert("L"), Image.open(after).convert("L")
        diff = ImageChops.difference(a, b).point(lambda v: 255 if v > 24 else 0)
        band = int(30 * s)
        diff.paste(0, (0, 0, diff.width, band))
        x0, y0, x1, y1 = diff.getbbox()
        return {"x": x0 / s, "y": height - y1 / s, "w": (x1 - x0) / s, "h": (y1 - y0) / s}

    # value: the Apple menu is the system's own menu on the same material as a
    # menu bar extra; opened twice per wall, the second take calibrates the noise.
    def native(self, scheme, wall, take):
        name = f"native-{scheme}-{wall}-{take}"
        self.appearance(scheme)
        self.place(wall, MENU_AT[0], MENU_AT[1])
        self.park()
        time.sleep(1)
        before = self.capture(f"{name}.closed")
        s, height = self.scale(None)
        self.wake()
        self.vnc.mouseMove(int(APPLE[0] * s), int(APPLE[1] * s))
        self.vnc.mousePress(1)
        time.sleep(1.5)
        self.park()
        time.sleep(0.5)
        full = self.capture(name)
        window = self.changed(before, full, None)
        self.vnc.keyPress("esc")
        time.sleep(1)
        rect = [int(round(window["x"] * s)), int(round((height - window["y"] - window["h"]) * s)),
                int(round((window["x"] + window["w"]) * s)), int(round((height - window["y"]) * s))]
        meta = {"name": name, "scheme": scheme, "wall": wall, "take": take, "rect_px": rect, "scale": s}
        self.say(f"-- {name}: {json.dumps(meta)}")
        return meta

    # value: the opening is read as the pictures the screen shows after the click, taken
    # as fast as VNC answers, cropped to the panel's columns; the app's own per-display-frame
    # log of the glass frame comes with them.
    def opening(self, scheme, wall):
        name = f"opening-{scheme}-{wall}"
        self.appearance(scheme)
        line = self.launch({"STATION_TRIAL_FRAMES": "1"})
        item = self.frame(line) if line else None
        if not item:
            self.say(f"FAIL {name}: the app logged no item frame")
            return None
        self.set_wall(wall, item)
        self.park()
        time.sleep(1)
        s, height = self.scale(item)
        left = max(item["x"] + item["w"] - WIDTH - 24, 0)
        box = (int(left * s), 0, min(int((left + WIDTH + 48) * s), self.fb[0]), self.fb[1])

        def grab():
            self.vnc.refreshScreen()
            return self.vnc.screen.convert("RGB").crop(box)

        self.reconnect()
        frames = [(0.0, grab())]
        t0 = time.time()
        self.click_item(item, hold=HOLD)
        while time.time() - t0 < SPAN:
            frames.append((time.time() - t0, grab()))
        self.park()
        folder = os.path.join(self.out, name)
        os.makedirs(folder, exist_ok=True)
        for i, (_, image) in enumerate(frames):
            image.save(os.path.join(folder, f"f{i:03d}.png"))
        log = self.ssh("cat ~/cap/app.log")
        with open(os.path.join(folder, "app.log"), "w", encoding="utf-8") as f:
            f.write(log)
        self.vnc.keyPress("esc")
        time.sleep(1)
        pill = [int(item["x"] * s) - box[0], int((item["x"] + item["w"]) * s) - box[0]]
        meta = {"name": name, "scheme": scheme, "wall": wall, "scale": s, "times": [round(t, 3) for t, _ in frames], "item_px": pill}
        self.say(f"-- {name}: {len(frames)} frames in {frames[-1][0]:.1f} s")
        return meta

    def behave(self, item):
        results = {}
        for gesture in ["item", "outside", "esc"]:
            self.click_item(item)
            time.sleep(2)
            opened = self.last("opened")
            if gesture == "item":
                self.click_item(item)
            elif gesture == "outside":
                self.vnc.mouseMove(40, self.fb[1] // 2)
                self.vnc.mousePress(1)
            else:
                self.vnc.keyPress("esc")
            time.sleep(1.5)
            log = self.ssh("cat ~/cap/app.log")
            results[gesture] = {"opened": opened.split(" ", 2)[-1] if opened else None, "closed": log.rstrip().endswith("closed")}
            self.say(f"-- behave {gesture}: {results[gesture]}")
        return results

    def trial(self, keyboard):
        self.appearance("dark", keyboard=keyboard)
        line = self.launch({"STATION_TRIAL": "1", "STATION_TRIAL_FPS": "0", "STATION_TRIAL_CYCLE": "1"})
        item = self.frame(line)
        self.click_item(item)
        time.sleep(4)
        self.park()
        for _ in range(3):
            self.vnc.keyPress("tab")
            time.sleep(0.4)
        self.vnc.keyPress("shift-tab")
        time.sleep(0.6)
        self.capture(f"trial-keyboard{keyboard}")
        log = self.ssh("cat ~/cap/app.log")
        with open(os.path.join(self.out, f"trial-keyboard{keyboard}.log"), "w", encoding="utf-8") as f:
            f.write(log)
        self.vnc.keyPress("esc")
        time.sleep(1)
        self.say(f"-- trial keyboard={keyboard}: {len(log.splitlines())} lines")


def main():
    ip, key, url, repo, out, sets = sys.argv[1:7]
    sets = sets.split(",")
    os.makedirs(out, exist_ok=True)
    rig = Rig(ip, key, url, repo, out)
    rig.setup()
    frames = {"STATION_TRIAL_FRAMES": "1"}
    shots = []
    old_item = None
    if "old" in sets:
        old_item = dict(zip("xywh", map(float, sys.argv[7].split(","))))
    if "behave" in sets:
        rig.appearance("dark")
        line = rig.launch(frames)
        if not line:
            rig.say("FAIL behave: the app logged no item frame")
            sys.exit(1)
        item = rig.frame(line)
        rig.set_wall("dark", item)
        behave = rig.behave(item)
        json.dump(behave, open(os.path.join(out, "behave.json"), "w"), indent=1)
    for scheme in SCHEMES:
        for wall in WALLS:
            if "walls" in sets:
                shots.append(rig.shot(f"{scheme}-{wall}", scheme, wall, frames))
            if "old" in sets:
                shots.append(rig.shot(f"old-{scheme}-{wall}", scheme, wall, {}, old_item=old_item))
            if "a11y" in sets:
                shots.append(rig.shot(f"{scheme}-{wall}-rt", scheme, wall, frames, reduce=True))
                shots.append(rig.shot(f"{scheme}-{wall}-ic", scheme, wall, frames, contrast=True))
    natives, openings = [], []
    if "native" in sets:
        rig.launch(frames)
        for scheme in SCHEMES:
            for wall in WALLS:
                for take in (1, 2):
                    natives.append(rig.native(scheme, wall, take))
        json.dump(natives, open(os.path.join(out, "native.json"), "w"), indent=1)
    if "opening" in sets:
        for scheme, wall in (("dark", "dark"), ("light", "light")):
            openings.append(rig.opening(scheme, wall))
        json.dump([o for o in openings if o], open(os.path.join(out, "opening.json"), "w"), indent=1)
    if "trial" in sets:
        for keyboard in (3, 0):
            rig.trial(keyboard)
    rig.appearance("dark")
    json.dump([s for s in shots if s], open(os.path.join(out, "shots.json"), "w"), indent=1)
    rig.vnc.disconnect()
    api.shutdown()
    failed = [s for s in shots + openings if s is None]
    rig.say(f"-- {len(shots) - len(failed)} shots, {len(failed)} failed")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
