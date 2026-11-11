#!/usr/bin/env python3
"""Build the WhatsLite doodle wallpaper master SVG and its previews.

The master is a 1024 square, edge-wrapped alpha tile.  Every motif is placed
once in the source file; copies that cross an edge are emitted at the opposite
edge so the tile can be repeated by the GPUI renderer without a visible seam.

Usage:
    python3 design/wallpaper/generate_doodle.py            # master + previews
    python3 design/wallpaper/generate_doodle.py --sheet     # motif contact sheet
"""

import argparse
import math
import random
import subprocess
from pathlib import Path

from motifs import MOTIFS, TINY

TILE = 1024
PRIMARY_STROKE = 2.05
FILLER_STROKE = 1.45
SVG_OPACITY = 0.5

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "design" / "machine"
WORK = ROOT / "design" / "wallpaper"


def defs():
    parts = ["<defs>"]
    for name, fn in MOTIFS.items():
        parts.append(f'<g id="m-{name}">{fn()}</g>')
    for name, fn in TINY.items():
        parts.append(f'<g id="t-{name}">{fn()}</g>')
    parts.append("</defs>")
    return "\n".join(parts)


def sheet():
    cols = 8
    cell_w, cell_h = 160, 186
    rows = math.ceil(len(MOTIFS) / cols)
    width = cols * cell_w
    height = rows * cell_h + 40
    items = []
    for index, (name, fn) in enumerate(MOTIFS.items()):
        col, row = index % cols, index // cols
        x = col * cell_w + 30
        y = row * cell_h + 20
        items.append(
            f'<g transform="translate({x} {y}) scale(1.12)" '
            f'stroke-width="{PRIMARY_STROKE / 1.12:.2f}">{fn()}</g>'
            f'<text x="{col * cell_w + 80}" y="{row * cell_h + 156}" '
            f'text-anchor="middle" font-family="Inter, sans-serif" font-size="11" '
            f'fill="#555">{name}</text>'
        )
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}"
 viewBox="0 0 {width} {height}" color="#1f1f1f">
<rect width="{width}" height="{height}" fill="#ffffff"/>
<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">
{''.join(items)}
</g></svg>'''


def weighted_size(rng):
    r = rng.random()
    if r < 0.04:
        return rng.uniform(116, 136)
    if r < 0.14:
        return rng.uniform(96, 118)
    if r < 0.30:
        return rng.uniform(76, 96)
    if r < 0.60:
        return rng.uniform(56, 76)
    if r < 0.85:
        return rng.uniform(40, 56)
    return rng.uniform(26, 40)


def torus_delta(a, b):
    d = abs(a - b) % TILE
    return min(d, TILE - d)


def best_position(rng, size, accepted, tries=110):
    radius = size * 0.52
    best, best_score = None, -1e9
    for _ in range(tries):
        # Centres may sit slightly outside the tile: the wrapped copy keeps the
        # opposite edge just as dense, which is what removes the seam bands.
        x = rng.uniform(-radius * 0.35, TILE + radius * 0.35)
        y = rng.uniform(-radius * 0.35, TILE + radius * 0.35)
        score = 1e9
        for ax, ay, ar in accepted:
            d = math.hypot(torus_delta(x, ax), torus_delta(y, ay)) - (radius + ar)
            if d < score:
                score = d
        if score > best_score:
            best_score, best = score, (x, y)
    return best, best_score


def wrap_uses(kind, index, x, y, size, rot, stroke, stretch=(1.0, 1.0), skew=0.0):
    scale = size / 100.0
    sx, sy = stretch
    radius = size * 0.74
    out = []
    for ox in (-TILE, 0, TILE):
        for oy in (-TILE, 0, TILE):
            cx, cy = x + ox, y + oy
            if cx + radius < 0 or cx - radius > TILE or cy + radius < 0 or cy - radius > TILE:
                continue
            out.append(
                f'<use href="#{kind}" transform="translate({cx:.2f} {cy:.2f}) '
                f'rotate({rot:.2f}) skewX({skew:.2f}) '
                f'scale({scale * sx:.5f} {scale * sy:.5f}) translate(-50 -50)" '
                f'stroke-width="{stroke / (scale * math.sqrt(sx * sy)):.3f}"/>'
            )
    return out


def compose(seed):
    rng = random.Random(seed)
    names = list(MOTIFS)
    rng.shuffle(names)
    accepted = []
    primary = []

    sizes = [weighted_size(rng) for _ in range(186)]
    sizes.sort(reverse=True)
    for i, size in enumerate(sizes):
        name = names[i % len(names)]
        if i % len(names) == 0:
            rng.shuffle(names)
        radius = size * 0.52
        x = y = None
        # Let some marks hug an existing illustration: clusters break the
        # blue-noise evenness without creating tangencies.
        if accepted and rng.random() < 0.12:
            ax, ay, ar = rng.choice(accepted)
            angle = rng.uniform(0, math.tau)
            distance = ar + radius + rng.uniform(3, 15)
            cx = ax + math.cos(angle) * distance
            cy = ay + math.sin(angle) * distance
            if all(math.hypot(torus_delta(cx, bx), torus_delta(cy, by)) >= radius + br + 4 for bx, by, br in accepted):
                x, y = cx, cy
        if x is None:
            (x, y), score = best_position(rng, size, accepted)
            if score < -4:
                continue
        accepted.append((x, y, radius))
        rot = rng.uniform(-26, 26) if rng.random() > 0.12 else rng.uniform(-52, 52)
        stretch = (rng.uniform(0.94, 1.06), rng.uniform(0.94, 1.06))
        primary.append((name, x, y, size, rot, stretch,
                        rng.uniform(-3.5, 3.5), PRIMARY_STROKE * rng.uniform(0.9, 1.1)))

    # A second, smaller pass fills the holes that the grid skips left behind.
    for _ in range(22):
        size = rng.uniform(25, 38)
        name = rng.choice(names)
        (x, y), score = best_position(rng, size, accepted, tries=150)
        if score < -4:
            continue
        radius = size * 0.52
        accepted.append((x, y, radius))
        primary.append((name, x, y, size, rng.uniform(-35, 35),
                        (rng.uniform(0.92, 1.08), rng.uniform(0.92, 1.08)),
                        rng.uniform(-4, 4), PRIMARY_STROKE * rng.uniform(0.9, 1.1)))

    # A third pass drops small objects into the pockets left by the grid of
    # larger illustrations, the way the reference fills its gaps.
    for _ in range(96):
        size = rng.uniform(22, 42)
        name = rng.choice(names)
        (x, y), score = best_position(rng, size, accepted, tries=240)
        if score < -5:
            continue
        radius = size * 0.52
        accepted.append((x, y, radius))
        primary.append((name, x, y, size, rng.uniform(-42, 42),
                        (rng.uniform(0.92, 1.08), rng.uniform(0.92, 1.08)),
                        rng.uniform(-5, 5), PRIMARY_STROKE * rng.uniform(0.88, 1.1)))

    # Tiny decorative marks use a different, simpler vocabulary.
    tiny_names = list(TINY)
    weights = {"dot": 7, "ring": 5, "star": 4, "heart": 3, "sparkle": 4,
               "flower": 2, "leaf": 1, "squiggle": 1, "note": 1, "moon": 1,
               "cloud": 1, "arrow": 1, "plus": 4, "pebble": 7}
    weights = [weights.get(name, 1) for name in tiny_names]
    tiny = []
    tiny_centers = []
    for _ in range(600):
        name = rng.choices(tiny_names, weights=weights, k=1)[0]
        size = rng.uniform(6, 16)
        radius = size * 0.5
        best, best_score = None, -1e9
        for _ in range(44):
            x = rng.uniform(0, TILE)
            y = rng.uniform(0, TILE)
            score = 1e9
            for px, py, pr in tiny_centers:
                d = math.hypot(torus_delta(x, px), torus_delta(y, py)) - (radius + pr)
                if d < score:
                    score = d
            for px, py, pr in accepted:
                d = math.hypot(torus_delta(x, px), torus_delta(y, py)) - (radius + pr)
                if d < score:
                    score = d
            if score > best_score:
                best_score, best = score, (x, y)
        x, y = best
        tiny_centers.append((x, y, radius))
        tiny.append((name, x, y, size, rng.uniform(-40, 40),
                     (rng.uniform(0.9, 1.1), rng.uniform(0.9, 1.1)),
                     rng.uniform(-6, 6), FILLER_STROKE * rng.uniform(0.88, 1.12)))

    uses = []
    for name, x, y, size, rot, stretch, skew, ink in primary:
        uses.extend(wrap_uses("m-" + name, name, x, y, size, rot, ink, stretch, skew))
    for name, x, y, size, rot, stretch, skew, ink in tiny:
        uses.extend(wrap_uses("t-" + name, name, x, y, size, rot, ink, stretch, skew))

    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{TILE}" height="{TILE}"
 viewBox="0 0 {TILE} {TILE}" color="#000000">
{defs()}
<g opacity="{SVG_OPACITY}" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">
{''.join(uses)}
</g>
</svg>'''
    return svg, primary, tiny


def render(svg_path, png_path, size, color=None, background=None):
    source = svg_path.read_text()
    if color:
        source = source.replace('color="#000000"', f'color="{color}"', 1)
    temp = WORK / "_render.svg"
    temp.write_text(source)
    cmd = ["magick", "-background", "none", "-density", "144", str(temp), "-resize", f"{size}x{size}"]
    if background:
        cmd = ["magick", "-background", "none", "-density", "144", str(temp), "-resize", f"{size}x{size}",
               "-background", background, "-flatten"]
    subprocess.run(cmd + [str(png_path)], check=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sheet", action="store_true")
    parser.add_argument("--seed", type=int, default=41)
    args = parser.parse_args()

    if args.sheet:
        out = WORK / "motif-sheet.svg"
        out.write_text(sheet())
        subprocess.run(["magick", "-background", "white", "-density", "144", str(out),
                        "-resize", "1280x", str(WORK / "motif-sheet.png")], check=True)
        print(out)
        return

    svg, primary, tiny = compose(args.seed)
    master = OUT / "wallpaper-doodle.svg"
    master.write_text(svg)
    print(f"master: {master}  primary={len(primary)} tiny={len(tiny)}")

    # Alpha masks used by the build scripts and the design handoff.
    render(master, OUT / "wallpaper-doodle.png", 1600)
    render(master, WORK / "tile-light.png", 1600, color="#C9B9A3", background="#F4F1EB")
    render(master, WORK / "tile-dark.png", 1600, color="#D8CCBC", background="#0B141A")

    # 3 x 3 seamless proof, at 432 px per repeat (1296 wide).
    tile = (WORK / "tile-light.png").resolve()
    subprocess.run([
        "magick", "-size", "1296x1296", "xc:#F4F1EB",
        "(", str(tile), "-resize", "432x432", ")",
        "-geometry", "+0+0", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+432+0", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+864+0", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+0+432", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+432+432", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+864+432", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+0+864", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+432+864", "-composite",
        "(", str(tile), "-resize", "432x432", ")", "-geometry", "+864+864", "-composite",
        str(WORK / "tile-3x3.png"),
    ], check=True)

    # Conversation-stage size proofs (940 x 736 at the app's 550 pt repeat).
    for name, tint, bg, opacity in (
        ("chat-light", "#C9B9A3", "#F4F1EB", 1.0),
        ("chat-dark", "#D8CCBC", "#0B141A", 0.12),
    ):
        render(master, WORK / "_mask.png", 1600)
        tile_path = WORK / f"_{name}-tile.png"
        subprocess.run([
            "magick", str(WORK / "_mask.png"), "-resize", "550x550",
            "-fill", tint, "-colorize", "100",
            "-alpha", "set", "-channel", "A", "-evaluate", "multiply", str(opacity), "+channel",
            str(tile_path),
        ], check=True)
        cmd = ["magick", "-size", "940x736", f"xc:{bg}"]
        for x, y in ((0, 0), (550, 0), (0, 550), (550, 550)):
            cmd += ["(", str(tile_path), ")", "-geometry", f"+{x}+{y}", "-composite"]
        cmd += [str(WORK / f"{name}.png")]
        subprocess.run(cmd, check=True)


if __name__ == "__main__":
    main()
