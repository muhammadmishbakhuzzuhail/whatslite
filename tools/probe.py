#!/usr/bin/env python3
# Geometry probe: measure structural dimensions from a sidebar render so parity
# is checked with NUMBERS, not eyeballing. Cross-engine renders (Chrome vs Qt)
# differ in font anti-aliasing, so raw pixel-diff is noisy — measured geometry
# (row pitch, first-row Y, avatar band) is the reliable signal.
#
#   tools/probe.py <sidebar.png> [avatar_x0] [avatar_x1]
import sys
import numpy as np
from PIL import Image

path = sys.argv[1]
x0 = int(sys.argv[2]) if len(sys.argv) > 2 else 76
x1 = int(sys.argv[3]) if len(sys.argv) > 3 else 104

im = np.asarray(Image.open(path).convert("RGB"))[:, x0:x1, :].astype(int)
col = im.mean(axis=1)
sat = col.max(axis=1) - col.min(axis=1)   # chroma → avatar circles are coloured
bright = col.mean(axis=1)
mask = (sat > 25) & (bright > 45)

ys = np.where(mask)[0]
centers, run = [], []
for y in ys:
    if run and y - run[-1] > 4:
        if len(run) > 12:
            centers.append(int(np.mean(run)))
        run = []
    run.append(y)
if len(run) > 12:
    centers.append(int(np.mean(run)))

pitch = [centers[i + 1] - centers[i] for i in range(len(centers) - 1)]
clean = [p for p in pitch if 55 < p < 90]   # drop merged/section gaps
med = int(np.median(clean)) if clean else 0
print(f"file        : {path}")
print(f"avatars     : {len(centers)}  first_y={centers[0] if centers else '-'}")
print(f"row pitch   : {clean}  median={med}px")
