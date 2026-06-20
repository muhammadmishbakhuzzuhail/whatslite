#!/usr/bin/env python3
# Cross-engine LAYOUT diff (Chrome vs Qt). A raw perceptual scalar (DSSIM/RMSE)
# is useless here: the two engines rasterise text with different anti-aliasing,
# so any text region reads as ~0.18 DSSIM even when the layout is pixel-identical
# — the font-AA noise swamps real regressions.
#
# Per cross-engine visual-testing practice, this compares STRUCTURE not pixels:
# grayscale -> Gaussian blur (collapses sub-pixel hinting/AA into the same
# low-frequency signal) -> coarse block grid -> per-block mean difference. A
# wrong position / wrong size / missing element / wrong colour survives the blur;
# font smoothing does not. Output: a 0..1 layout-diff score (lower = closer) and
# a block heatmap of WHERE the structure differs.
#
#   tools/layout-diff.py A.png B.png [out-heatmap.png] [grid_cols]
import sys
import numpy as np
from PIL import Image, ImageFilter

A, B = sys.argv[1], sys.argv[2]
OUT = sys.argv[3] if len(sys.argv) > 3 else "/tmp/layout-heat.png"
COLS = int(sys.argv[4]) if len(sys.argv) > 4 else 55   # block columns; rows scaled to aspect


def prep(path, w, h):
    im = Image.open(path).convert("L").resize((w, h))
    im = im.filter(ImageFilter.GaussianBlur(radius=2.5))   # kill font AA, keep layout
    return np.asarray(im, dtype=np.float32)


# normalise both to A's size
aw, ah = Image.open(A).size
rows = max(1, round(COLS * ah / aw))
# work at a coarse block resolution (downsample = the AA-ignoring step)
ga = prep(A, COLS, rows)
gb = prep(B, COLS, rows)

diff = np.abs(ga - gb)                       # per-block mean-intensity difference
score = float(diff.mean() / 255.0)           # 0 = identical layout, higher = more drift
worst = np.unravel_index(np.argsort(diff, axis=None)[-5:], diff.shape)

print(f"layout-diff score : {score:.4f}   (0=identical structure; font-AA-immune)")
print(f"blocks            : {COLS}x{rows}   max-block-delta={diff.max()/255.0:.3f}")
# report the worst-differing block bands (as fractions of the image) to localise drift
ys, xs = worst
print("worst blocks (x%,y%):", ", ".join(
    f"({int(100*x/COLS)},{int(100*y/rows)})" for y, x in sorted(zip(ys, xs))))

# heatmap: red where blocks differ, upscaled back to A's size, over a dim B
heat = (diff / max(1e-6, diff.max()) * 255).astype(np.uint8)
heat_img = Image.fromarray(heat).resize((aw, ah), Image.NEAREST)
red = Image.merge("RGB", (heat_img,
                          Image.new("L", (aw, ah), 0),
                          Image.new("L", (aw, ah), 0)))
base = Image.open(B).convert("RGB").resize((aw, ah))
Image.blend(base, red, 0.5).save(OUT)
print(f"heatmap           : {OUT}")
