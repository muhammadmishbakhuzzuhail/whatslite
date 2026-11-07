#!/usr/bin/env python3
"""Gap-filled seamless doodle tile, packed with stroke masks.

Large -> medium -> small.  Small marks are real icons from the same pool, not
only dots.  A swept raster mask guarantees that no two strokes cross; bounding
boxes are allowed to interlock, exactly like the reference.
"""

import math
import random
import subprocess
import sys
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFilter

sys.path.insert(0, str(Path(__file__).parent))
import build_source_wallpaper as base  # noqa: E402

TILE = 1024
STROKE = 2.05
SEED = 91
PRIMARY = 300
PANEL_W, PANEL_H, PANEL_SCALE, REPEAT = 636, 800, 2, 800
ROOT = Path(__file__).resolve().parents[2]
WORK = ROOT / "design" / "wallpaper"
MACHINE = ROOT / "design" / "machine"
PEN = Path("/home/muhammadmishbakhuzzuhail/.pencil/documents/3ed9f4dc-de41-4734-b584-f2b77e7dc12f")
CACHE = Path("/tmp/doodle-masks")
CACHE.mkdir(exist_ok=True)


def base_mask(source, name, vb, body):
    path = CACHE / f"{source}{name}.png"
    if not path.exists():
        sw = STROKE * vb / 512.0
        svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 {vb} {vb}" color="#000" fill="none" stroke="#000" stroke-linecap="round" stroke-linejoin="round">
<g stroke-width="{sw:.4f}">{body}</g></svg>'''
        tmp = CACHE / f"{source}{name}.svg"
        tmp.write_text(svg)
        subprocess.run(["magick", "-background", "none", str(tmp), str(path)], check=True)
    img = Image.open(path).convert("RGBA")
    return img.getchannel("A")


def base_fill(source, name, vb, body):
    path = CACHE / f"{source}{name}-fill.png"
    if not path.exists():
        svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 {vb} {vb}" color="#000" fill="#000" stroke="none">
<g fill="#000" stroke="none">{body}</g></svg>'''
        tmp = CACHE / f"{source}{name}-fill.svg"
        tmp.write_text(svg)
        subprocess.run(["magick", "-background", "none", str(tmp), str(path)], check=True)
    img = Image.open(path).convert("RGBA")
    return img.getchannel("A")


def make_mask(base_img, size, rot, pad=1):
    s = max(1, int(round(size)))
    m = base_img.resize((s, s), Image.LANCZOS)
    m = m.point(lambda p: 255 if p > 18 else 0)
    m = m.filter(ImageFilter.MaxFilter(2 * pad + 1))  # stroke safety margin
    if abs(rot) > 0.01:
        m = m.rotate(-rot, expand=True, resample=Image.BICUBIC)
        m = m.point(lambda p: 255 if p > 18 else 0)
    return m


def stamp(occ, mask, x, y, check=True):
    w, h = mask.size
    left = int(round(x - w / 2))
    top = int(round(y - h / 2))
    boxes = []
    for ox in (-TILE, 0, TILE):
        for oy in (-TILE, 0, TILE):
            bx, by = left + ox, top + oy
            x0, y0 = max(0, bx), max(0, by)
            x1, y1 = min(TILE, bx + w), min(TILE, by + h)
            if x1 <= x0 or y1 <= y0:
                continue
            mc = mask.crop((x0 - bx, y0 - by, x1 - bx, y1 - by))
            if check:
                if ImageChops.multiply(occ.crop((x0, y0, x1, y1)), mc).getbbox():
                    return False
            boxes.append((x0, y0, mc))
    for x0, y0, mc in boxes:
        occ.paste(255, (x0, y0), mc)
    return True


def main():
    rng = random.Random(SEED)
    pool = base.load_pool()
    accents = base.load_accents()
    print("pool", len(pool), "accents", len(accents))

    bases = {}
    fills = {}
    for source, name, vb, body in pool:
        bases[(source, name)] = base_mask(source, name, vb, body)
        fills[(source, name)] = base_fill(source, name, vb, body)
    for source, name, vb, body in accents:
        bases[(source, name)] = base_mask(source, name, vb, body)
        fills[(source, name)] = base_fill(source, name, vb, body)

    occ = Image.new("L", (TILE, TILE), 0)
    area = Image.new("L", (TILE, TILE), 0)
    placements = []
    small_centers = []
    dot_centers = []

    def stamp_fill(source, name, size, x, y, rot):
        f = fills[(source, name)].resize((max(1, int(round(size))),) * 2, Image.LANCZOS)
        f = f.point(lambda p: 255 if p > 40 else 0)
        f = f.filter(ImageFilter.MaxFilter(3))
        if abs(rot) > 0.01:
            f = f.rotate(-rot, expand=True, resample=Image.BICUBIC)
            f = f.point(lambda p: 255 if p > 40 else 0)
        stamp(area, f, x, y, check=False)

    def place(source, name, vb, body, size, x, y, rot, primary=False, pad=1):
        mask = make_mask(bases[(source, name)], size, rot, pad)
        if not stamp(occ, mask, x, y, check=True):
            return False
        placements.append((source, name, vb, body, size, x, y, rot))
        if primary:
            stamp_fill(source, name, size, x, y, rot)
        return True

    def blocked_by_primary(mask, x, y):
        # Only the centre is protected, so marks may sit between the strokes
        # of a large icon; the stroke mask still stops them from crossing it.
        return area.getpixel((int(round(x)) % TILE, int(round(y)) % TILE)) > 0

    def too_close(x, y, radius, centers):
        for cx, cy, cr in centers:
            if math.hypot(base.td(x, cx), base.td(y, cy)) < radius + cr:
                return True
        return False

    # ---- 330 primary objects, largest first, no stroke overlap
    tiers = [(92, 105, 5), (74, 92, 25), (56, 74, 100), (40, 56, 120), (24, 40, 50)]
    sizes = []
    for lo, hi, count in tiers:
        sizes += [rng.uniform(lo, hi) for _ in range(count)]
    sizes.sort(reverse=True)

    names = pool[:]
    rng.shuffle(names)
    heroes = [p for p in pool if p[1] in base.HERO]
    rng.shuffle(heroes)
    hero_i = 0
    placed_primary = 0
    for i, size in enumerate(sizes):
        source, name, vb, body = (
            heroes[hero_i % len(heroes)] if size >= 85 and heroes else names[i % len(names)]
        )
        if size >= 74 and heroes:
            hero_i += 1
        ok = False
        for _ in range(90):
            x = rng.uniform(0, TILE)
            y = rng.uniform(0, TILE)
            rot = rng.uniform(-24, 24) if rng.random() > 0.15 else rng.uniform(-50, 50)
            if place(source, name, vb, body, size, x, y, rot, primary=True):
                ok = True
                break
        placed_primary += 1 if ok else 0
    print("primary", placed_primary, "of", len(sizes))

    # ---- small icons: peel the largest empty circles, then smaller ones.
    small_count = 0
    for r in (20, 16, 13, 10):
        dil = occ.filter(ImageFilter.MaxFilter(2 * r + 1))
        step = max(5, r)
        pts = [(x, y) for y in range(step // 2, TILE, step) for x in range(step // 2, TILE, step)
               if dil.getpixel((x, y)) == 0]
        rng.shuffle(pts)
        added = 0
        for x0, y0 in pts:
            for _ in range(4):
                source, name, vb, body = rng.choice(pool)
                size = max(16, min(36, 2 * (r - 2) + rng.uniform(-2, 2)))
                x = x0 + rng.uniform(-step * 0.45, step * 0.45)
                y = y0 + rng.uniform(-step * 0.45, step * 0.45)
                rot = rng.uniform(-55, 55)
                mask = make_mask(bases[(source, name)], size, rot, 2)
                radius = size * 0.5
                if area.getpixel((int(round(x)) % TILE, int(round(y)) % TILE)) > 0:
                    continue
                if too_close(x, y, radius + 5, small_centers):
                    continue
                if place(source, name, vb, body, size, x, y, rot, pad=2):
                    small_centers.append((x, y, radius))
                    small_count += 1
                    added += 1
                    break
        print("small r", r, "added", added, "total", small_count)

    # ---- cell audit: every cell that has no small-icon centre gets one
    for cell in (32, 24, 18):
        added = 0
        occupied = set()
        for cx, cy, _ in small_centers:
            occupied.add((int(cx) // cell, int(cy) // cell))
        for y in range(cell // 2, TILE, cell):
            for x in range(cell // 2, TILE, cell):
                if (x // cell, y // cell) in occupied:
                    continue
                if area.getpixel((x % TILE, y % TILE)) > 0:
                    continue
                for _ in range(4):
                    source, name, vb, body = rng.choice(pool)
                    size = rng.uniform(14, max(14.5, min(30, cell * 0.82)))
                    xx = x + rng.uniform(-cell * 0.22, cell * 0.22)
                    yy = y + rng.uniform(-cell * 0.22, cell * 0.22)
                    rot = rng.uniform(-55, 55)
                    mask = make_mask(bases[(source, name)], size, rot, 2)
                    if too_close(xx, yy, size * 0.5 + 5, small_centers):
                        continue
                    if place(source, name, vb, body, size, xx, yy, rot, pad=2):
                        small_centers.append((xx, yy, size * 0.5))
                        occupied.add((int(xx) // cell, int(yy) // cell))
                        small_count += 1
                        added += 1
                        break
        print("cell", cell, "added", added, "total", small_count)

    # ---- dots in whatever is left
    dot_count = 0
    dot_sizes = (12, 10, 8, 6)
    max_dots = 520
    points = [(x, y) for y in range(3, TILE, 6) for x in range(3, TILE, 6)]
    rng.shuffle(points)
    for x0, y0 in points:
        if dot_count >= max_dots:
            break
        source, name, vb, body = rng.choice(accents)
        size = rng.choice(dot_sizes)
        x = x0 + rng.uniform(-4, 4)
        y = y0 + rng.uniform(-4, 4)
        mask = make_mask(bases[(source, name)], size, 0, 2)
        if blocked_by_primary(mask, x, y):
            continue
        if too_close(x, y, size * 0.5 + 6, small_centers):
            continue
        if too_close(x, y, size * 0.5 + 4, dot_centers):
            continue
        if place(source, name, vb, body, size, x, y, rng.uniform(-60, 60), pad=2):
            dot_centers.append((x, y, size * 0.5))
            dot_count += 1
    print("dots", dot_count, "total placements", len(placements))

    # ---- build tile SVG
    uses = []
    for source, name, vb, body, size, x, y, rot in placements:
        scale = size / vb
        radius = size * 0.85
        for ox in (-TILE, 0, TILE):
            for oy in (-TILE, 0, TILE):
                cx, cy = x + ox, y + oy
                if cx + radius < 0 or cx - radius > TILE or cy + radius < 0 or cy - radius > TILE:
                    continue
                uses.append(
                    f'<g transform="translate({cx:.2f} {cy:.2f}) rotate({rot:.2f}) '
                    f'scale({scale:.5f}) translate({-vb / 2:.1f} {-vb / 2:.1f})" '
                    f'stroke-width="{STROKE / scale:.3f}">{body}</g>'
                )
    tile_group = (
        '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round">'
        + "".join(uses)
        + "</g>"
    )
    master = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{TILE}" height="{TILE}" viewBox="0 0 {TILE} {TILE}" color="#000000">
{tile_group}
</svg>'''
    (MACHINE / "wallpaper-doodle-source.svg").write_text(master)

    def panel(bg, ink, opacity, name):
        s = (REPEAT * PANEL_SCALE) / TILE
        w, h = PANEL_W * PANEL_SCALE, PANEL_H * PANEL_SCALE
        tile = REPEAT * PANEL_SCALE
        copies = []
        for r in range(math.ceil(h / tile) + 1):
            for c in range(math.ceil(w / tile) + 1):
                copies.append(f'<g transform="translate({c * tile} {r * tile}) scale({s:.6f})">{tile_group}</g>')
        svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" color="{ink}">
<rect width="{w}" height="{h}" fill="{bg}"/>
<g opacity="{opacity}">{''.join(copies)}</g>
</svg>'''
        sp = WORK / f"_{name}.svg"
        sp.write_text(svg)
        subprocess.run(["magick", "-background", "none", str(sp), str(WORK / f"{name}.png")], check=True)
        (PEN / f"wp38-{name}.png").write_bytes((WORK / f"{name}.png").read_bytes())
        print("panel", name, w, h)

    panel("#F4F1EB", "#C9B9A3", 0.60, "seamless-light-v19")
    panel("#0B141A", "#D8CCBC", 0.20, "seamless-dark-v19")
    print("COUNTS", placed_primary, small_count, dot_count)


if __name__ == "__main__":
    main()
