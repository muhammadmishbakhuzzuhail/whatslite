#!/usr/bin/env python3
"""Compose the WhatsLite doodle wall from the two chosen MIT sources.

Objects : Sketchy Icon Library (MIT) + Oreo Doodle Icons (MIT)
Accents : tiny marks from the same two sets
Master  : 1024 x 1024 seamless, wrapped on all four edges
Preview : 636 x 900 @ 2x, light 60% / dark 20%
"""

import math
import random
import re
import subprocess
from pathlib import Path

TILE = 1024
PANEL_W, PANEL_H = 636, 900
PANEL_SCALE = 2
REPEAT = 800
OBJECT_STROKE = 2.05
ACCENT_STROKE = 1.25
SEED = 73

SKETCHY = Path("/tmp/sketchy-icons/svg")
OREO = Path("/tmp/oreo-icons/icons")
ROOT = Path(__file__).resolve().parents[2]
WORK = ROOT / "design" / "wallpaper"
MACHINE = ROOT / "design" / "machine"
PEN = Path("/home/muhammadmishbakhuzzuhail/.pencil/documents/3ed9f4dc-de41-4734-b584-f2b77e7dc12f")

# Deliberately curated: everyday objects, food, nature, travel, people,
# celebration and home. UI verbs, finance, charts, logos and arrows are out.
SKETCHY_NAMES = """
chat chat-bubble comment message speech-bubble envelope email mail-open mailbox
phone smartphone laptop camera webcam headphones microphone
podcast robot game-controller joystick tablet battery
battery-tech satellite telescope rocket planet spaceship comet galaxy atom
microscope video-call
apple-fruit banana bread burger cake cheese coffee cookie cupcake donut egg ice-cream
milk pizza salad soup wine beer cup water-glass
cactus flower leaf leaves mushroom pine-tree plant seedling tree mountain rainbow sun
moon-weather sun-nature sun-behind-cloud partly-cloudy cloud-nature cloud-rain rain
snow snowflake snowman storm tornado-weather wind drop dew sunrise sunset lightning
hail fog-weather thermometer-weather
airplane bicycle bike boat bus car ferry helicopter hot-air-balloon motorcycle scooter
ship spaceship submarine train van taxi tent hotel passport boarding-pass suitcase map
compass atlas globe location-pin lighthouse castle church school-building hospital-building
office-building apartment house barn windmill road-sign
traffic-light tire
birthday celebrate present king queen trophy-sport star heart favorite diamond-shape
ring-shape love couple family
key lock unlock bulb umbrella clock alarm-clock wristwatch bottle backpack shopping-bag
wallet bell book ruler scissors-tool paint-brush desk table brush pencil pen
notebook paperclip calendar basket
happy sad angry laughing crying person man woman baby doctor nurse scientist artist
teacher student farmer worker pilot engineer crowd team people
ram anchor magnet hourglass sandglass flag target
""".split()

OREO_NAMES = """
autumn-tree balloons bat bird birthday-candle book book-open brush bulb butterfly calendar
camera candy-cane car cat cherry-blossom clouds confetti corn cupcake daisy diamond dog
fire fish flag flame flower gamepad ghost gift-bag gingerbread-man globe heart hearts
heart-arrow home key laptop leaf love-letter love-lock mail maple-leaf moon moon-stars
party-hat paw-print pencil pen phone pie-slice potted-plant present pumpkin pumpkin-pie
rabbit rain ring rose ruler scissors shooting-star smile frown snowflake snowman
sparkle-heart spider star star-medal stars sun sunflower tree-pine tulip umbrella videocam
""".split()

SKETCHY_ACCENTS = """
star heart flower leaf drop circle-shape ring-shape oval diamond-shape
""".split()

OREO_ACCENTS = """
star stars sparkle-heart hearts heart diamond flower daisy leaf ring circle
""".split()

ACCENT_WEIGHT = {
    "circle-shape": 10, "circle": 10, "ring-shape": 9, "star": 8, "stars": 6,
    "heart": 6, "hearts": 4, "diamond": 6, "diamond-shape": 6, "flower": 5,
    "daisy": 4, "leaf": 4, "sparkle-heart": 3, "drop": 4, "oval": 4,
    "sun-nature": 2, "moon-weather": 2, "cloud-nature": 2, "snowflake": 2,
    "confetti": 3, "shooting-star": 2, "moon": 2, "sun": 2, "clouds": 2,
    "rain": 2, "cherry-blossom": 2, "tulip": 2, "rose": 2, "sunflower": 2,
}


def read_icon(source, name):
    path = (SKETCHY if source == "s" else OREO) / f"{name}.svg"
    if not path.exists():
        return None
    raw = path.read_text()
    if source == "s":
        m = re.search(r'<g class="ink"[^>]*>([\s\S]*)</g>\s*</svg>', raw)
        if not m:
            return None
        body = m.group(1)
        vb = 96.0
    else:
        m = re.search(r"<svg[^>]*>([\s\S]*)</svg>", raw)
        if not m:
            return None
        body = m.group(1)
        vb = 48.0
    # Namespace any ids so repeated icons cannot reference each other.
    ids = re.findall(r'id="([^"]+)"', body)
    for i, ident in enumerate(ids):
        body = body.replace(f'id="{ident}"', f'id="{source}{name}{i}"')
        body = body.replace(f"url(#{ident})", f"url(#{source}{name}{i})")
    body = re.sub(r'stroke-width="[^"]*"', "", body)
    body = re.sub(r'stroke="[^"]*"', "", body)
    body = re.sub(r'fill="[^"]*"', "", body)
    body = re.sub(r'style="[^"]*"', "", body)
    return vb, body


def load_pool():
    pool = []
    for name in SKETCHY_NAMES:
        icon = read_icon("s", name)
        if icon:
            pool.append(("s", name, *icon))
    for name in OREO_NAMES:
        icon = read_icon("o", name)
        if icon:
            pool.append(("o", name, *icon))
    return pool


def load_accents():
    accents = []
    for name in SKETCHY_ACCENTS:
        icon = read_icon("s", name)
        if icon:
            accents.append(("s", name, *icon))
    for name in OREO_ACCENTS:
        icon = read_icon("o", name)
        if icon:
            accents.append(("o", name, *icon))
    return accents


def td(a, b):
    d = abs(a - b) % TILE
    return min(d, TILE - d)


def best_position(rng, radius, accepted, tries=90, gap=0.0):
    best, best_score = None, -1e9
    for _ in range(tries):
        x = rng.uniform(-radius * 0.4, TILE + radius * 0.4)
        y = rng.uniform(-radius * 0.4, TILE + radius * 0.4)
        score = 1e9
        for ax, ay, ar in accepted:
            d = math.hypot(td(x, ax), td(y, ay)) - (radius + ar + gap)
            if d < score:
                score = d
        if score > best_score:
            best_score, best = score, (x, y)
    return best, best_score


def uses_for(kind, x, y, size, rot, vb, body, stroke, radius_factor=0.72):
    scale = size / vb
    radius = size * radius_factor
    out = []
    for ox in (-TILE, 0, TILE):
        for oy in (-TILE, 0, TILE):
            cx, cy = x + ox, y + oy
            if cx + radius < 0 or cx - radius > TILE or cy + radius < 0 or cy - radius > TILE:
                continue
            out.append(
                f'<g transform="translate({cx:.2f} {cy:.2f}) rotate({rot:.2f}) '
                f'scale({scale:.5f}) translate({-vb / 2:.1f} {-vb / 2:.1f})" '
                f'stroke-width="{stroke / scale:.3f}">{body}</g>'
            )
    return out


HERO = set("""
bicycle bike boat car train airplane rocket planet spaceship camera robot pizza burger
ice-cream cake coffee cat dog bird fish rabbit flower tree sun moon rainbow house present
balloons heart star smile phone laptop headphones microphone game-controller map compass
suitcase apple-fruit banana cheese donut cupcake
""".split())


def compose():
    rng = random.Random(SEED)
    pool = load_pool()
    accents = load_accents()
    print("pool", len(pool), "accents", len(accents))

    tiers = [(108, 135, 8), (85, 108, 30), (61, 85, 130), (40, 61, 135), (25, 40, 57)]
    sizes = []
    for lo, hi, count in tiers:
        sizes += [rng.uniform(lo, hi) for _ in range(count)]
    sizes.sort(reverse=True)

    names = pool[:]
    rng.shuffle(names)
    heroes = [p for p in pool if p[1] in HERO]
    rng.shuffle(heroes)
    hero_i = 0
    accepted = []
    object_uses = []
    for i, size in enumerate(sizes):
        if i % len(names) == 0:
            rng.shuffle(names)
        if size >= 85 and heroes:
            if hero_i >= len(heroes):
                rng.shuffle(heroes)
                hero_i = 0
            source, name, vb, body = heroes[hero_i]
            hero_i += 1
        else:
            source, name, vb, body = names[i % len(names)]
        radius = size * 0.52
        x = y = None
        if accepted and rng.random() < 0.35:
            ax, ay, ar = rng.choice(accepted)
            angle = rng.uniform(0, math.tau)
            distance = ar + radius + rng.uniform(-4, 10)
            cx = ax + math.cos(angle) * distance
            cy = ay + math.sin(angle) * distance
            if all(math.hypot(td(cx, bx), td(cy, by)) >= radius + br - 2 for bx, by, br in accepted):
                x, y = cx, cy
        if x is None:
            (x, y), _ = best_position(rng, radius, accepted)
        accepted.append((x, y, radius))
        rot = rng.uniform(-24, 24) if rng.random() > 0.14 else rng.uniform(-50, 50)
        object_uses += uses_for(source + name, x, y, size, rot, vb, body, OBJECT_STROKE)

    weights = [ACCENT_WEIGHT.get(name, 1) for _, name, _, _ in accents]
    accent_uses = []
    accent_centers = []
    for i in range(750):
        source, name, vb, body = rng.choices(accents, weights=weights, k=1)[0]
        if i < 450:
            size = rng.uniform(6, 10)
        elif i < 675:
            size = rng.uniform(10, 14)
        else:
            size = rng.uniform(14, 18)
        radius = size * 0.5
        (x, y), _ = best_position(rng, radius, accepted + accent_centers, tries=36, gap=-1)
        accent_centers.append((x, y, radius))
        rot = rng.uniform(-45, 45)
        accent_uses += uses_for(source + name, x, y, size, rot, vb, body, ACCENT_STROKE, 0.55)

    tile_group = (
        '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">'
        + "".join(object_uses)
        + "".join(accent_uses)
        + "</g>"
    )
    print("objects", len(sizes), "accents", 750, "uses", len(object_uses) + len(accent_uses))
    return tile_group


def panel(tile_group, bg, ink, opacity, out):
    scale = (REPEAT * PANEL_SCALE) / TILE
    w, h = PANEL_W * PANEL_SCALE, PANEL_H * PANEL_SCALE
    tile = REPEAT * PANEL_SCALE
    copies = []
    rows = math.ceil(h / tile) + 1
    cols = math.ceil(w / tile) + 1
    for r in range(rows):
        for c in range(cols):
            copies.append(f'<g transform="translate({c * tile} {r * tile}) scale({scale:.6f})">{tile_group}</g>')
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" color="{ink}">
<rect width="{w}" height="{h}" fill="{bg}"/>
<g opacity="{opacity}">{''.join(copies)}</g>
</svg>'''
    sp = WORK / f"_{out}.svg"
    sp.write_text(svg)
    subprocess.run(["magick", "-background", "none", str(sp), str(WORK / f"{out}.png")], check=True)
    print(out, w, h)


def main():
    tile_group = compose()

    master = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{TILE}" height="{TILE}" viewBox="0 0 {TILE} {TILE}" color="#000000">
{tile_group}
</svg>'''
    (MACHINE / "wallpaper-doodle-source.svg").write_text(master)

    panel(tile_group, "#F4F1EB", "#C9B9A3", 0.60, "seamless-light-v5")
    panel(tile_group, "#0B141A", "#D8CCBC", 0.20, "seamless-dark-v5")

    # Local 3x3 seam check at 432 per repeat.
    scale = 432 / TILE
    copies = []
    for r in range(3):
        for c in range(3):
            copies.append(f'<g transform="translate({c * 432} {r * 432}) scale({scale:.6f})">{tile_group}</g>')
    proof = f'''<svg xmlns="http://www.w3.org/2000/svg" width="1296" height="1296" viewBox="0 0 1296 1296" color="#C9B9A3">
<rect width="1296" height="1296" fill="#F4F1EB"/><g opacity="0.60">{''.join(copies)}</g></svg>'''
    (WORK / "_3x3.svg").write_text(proof)
    subprocess.run(["magick", "-background", "none", str(WORK / "_3x3.svg"), str(WORK / "v5-3x3.png")], check=True)

    for name in ("seamless-light-v5", "seamless-dark-v5"):
        src = WORK / f"{name}.png"
        (PEN / f"wp38-{name}.png").write_bytes(src.read_bytes())
        print("copied", PEN / f"wp38-{name}.png")


if __name__ == "__main__":
    main()
