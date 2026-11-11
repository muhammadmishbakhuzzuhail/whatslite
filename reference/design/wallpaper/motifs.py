"""Original monoline doodle vocabulary for the WhatsLite wallpaper.

Every motif is drawn inside a 100 x 100 box. Coordinates deliberately avoid
perfect symmetry: the wallpaper should read as hand-drawn, not as an icon set.
All colour is inherited from the root group (currentColor); the master is an
alpha mask and the app tints it at runtime.
"""

import math


def path(d, **attrs):
    extra = "".join(f' {k.replace("_", "-")}="{v}"' for k, v in attrs.items())
    return f'<path d="{d}"{extra}/>'


def circle(cx, cy, r, **attrs):
    extra = "".join(f' {k.replace("_", "-")}="{v}"' for k, v in attrs.items())
    return f'<circle cx="{cx:.2f}" cy="{cy:.2f}" r="{r:.2f}"{extra}/>'


def ellipse(cx, cy, rx, ry, rot=0, **attrs):
    extra = "".join(f' {k.replace("_", "-")}="{v}"' for k, v in attrs.items())
    transform = f' transform="rotate({rot} {cx:.2f} {cy:.2f})"' if rot else ""
    return f'<ellipse cx="{cx:.2f}" cy="{cy:.2f}" rx="{rx:.2f}" ry="{ry:.2f}"{transform}{extra}/>'


def rect(x, y, w, h, rx=0, **attrs):
    extra = "".join(f' {k.replace("_", "-")}="{v}"' for k, v in attrs.items())
    return f'<rect x="{x:.2f}" y="{y:.2f}" width="{w:.2f}" height="{h:.2f}" rx="{rx:.2f}"{extra}/>'


def line(x1, y1, x2, y2, **attrs):
    extra = "".join(f' {k.replace("_", "-")}="{v}"' for k, v in attrs.items())
    return f'<line x1="{x1:.2f}" y1="{y1:.2f}" x2="{x2:.2f}" y2="{y2:.2f}"{extra}/>'


def poly(points, closed=False, **attrs):
    d = "M" + " L".join(f"{x:.2f} {y:.2f}" for x, y in points) + (" Z" if closed else "")
    return path(d, **attrs)


def star(cx, cy, r1, r2=None, n=5, rot=-90, **attrs):
    r2 = r1 * 0.42 if r2 is None else r2
    pts = []
    for i in range(2 * n):
        r = r1 if i % 2 == 0 else r2
        a = math.radians(rot + i * 180 / n)
        pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    return poly(pts, True, **attrs)


def dot(cx, cy, r=3.0):
    return circle(cx, cy, r, fill="currentColor", stroke="none")


def j(*parts):
    return "".join(parts)


# ---------------------------------------------------------------- objects

def chat():
    return j(
        path("M18 34 Q18 20 32 20 L68 20 Q82 20 82 34 L82 58 Q82 70 68 70 L52 70 L42 86 L38 70 L32 70 Q18 70 18 58 Z"),
        dot(35, 45, 2.7), dot(50, 45, 2.7), dot(65, 45, 2.7),
    )


def chat_heart():
    return j(
        path("M18 34 Q18 20 32 20 L68 20 Q82 20 82 34 L82 58 Q82 70 68 70 L52 70 L42 86 L38 70 L32 70 Q18 70 18 58 Z"),
        path("M50 58 C42 53 39 47 42 43 C45 39 50 41 50 45 C50 41 55 39 58 43 C61 47 58 53 50 58 Z"),
    )


def heart():
    return path("M50 84 C22 66 13 48 22 34 C31 21 47 25 50 38 C53 25 69 21 78 34 C87 48 78 66 50 84 Z")


def star5():
    return star(50, 51, 35, 16)


def smile():
    return j(circle(50, 50, 35), dot(38, 44, 3.2), dot(62, 44, 3.2), path("M34 58 Q50 74 66 58"))


def wink():
    return j(circle(50, 50, 35), dot(36, 44, 3.2), line(58, 44, 68, 44), path("M33 58 Q49 72 67 57"))


def flower():
    petals = "".join(
        f'<ellipse cx="50" cy="28" rx="9" ry="14" transform="rotate({a} 50 50)"/>'
        for a in range(0, 360, 60)
    )
    return j(petals, circle(50, 50, 7), circle(50, 50, 2.5))


def leaf():
    return j(
        path("M50 12 C74 24 84 50 50 88 C16 50 26 24 50 12 Z"),
        path("M50 20 L50 80"),
        path("M50 38 L66 30"), path("M50 52 L68 45"), path("M50 64 L64 60"),
        path("M50 38 L34 30"), path("M50 52 L32 45"),
    )


def cloud():
    return path("M22 69 C10 66 8 50 20 46 C18 31 34 23 45 32 C51 17 74 17 80 33 C94 33 97 52 84 59 C86 70 74 77 63 71 C55 79 33 79 27 71 Z")


def sun():
    rays = "".join(
        line(50 + 25 * math.cos(math.radians(a)), 50 + 25 * math.sin(math.radians(a)),
             50 + 37 * math.cos(math.radians(a)), 50 + 37 * math.sin(math.radians(a)))
        for a in range(0, 360, 45)
    )
    return j(circle(50, 50, 18), rays)


def moon():
    return path("M65 14 A36 36 0 1 0 65 86 A28 28 0 1 1 65 14 Z")


def planet():
    return j(
        circle(50, 50, 20),
        ellipse(50, 50, 37, 10, rot=-18),
        circle(78, 31, 4),
    )


def rocket():
    return j(
        path("M50 8 C62 22 66 42 66 60 L34 60 C34 42 38 22 50 8 Z"),
        circle(50, 38, 8),
        path("M34 48 L21 66 L34 64 Z"),
        path("M66 48 L79 66 L66 64 Z"),
        path("M42 65 L50 86 L58 65"),
        path("M46 65 L50 76 L54 65"),
        line(26, 42, 20, 36), line(24, 52, 16, 48),
    )


def plane():
    return j(
        path("M10 52 L90 16 L62 90 L47 61 Z"),
        path("M10 52 L47 61 L90 16"),
        path("M47 61 L42 82"),
    )


def envelope():
    return j(
        rect(12, 26, 76, 50, 5),
        path("M12 30 L50 57 L88 30"),
        path("M12 74 L40 52"), path("M88 74 L60 52"),
    )


def pencil():
    return j(
        rect(38, 8, 24, 58, 3),
        path("M38 66 L50 90 L62 66"),
        line(38, 18, 62, 18),
        path("M42 24 L42 62"), path("M58 24 L58 62"),
    )


def book():
    return j(
        path("M50 24 C38 15 22 15 12 21 L12 76 C24 70 38 70 50 78 C62 70 76 70 88 76 L88 21 C78 15 62 15 50 24 Z"),
        line(50, 24, 50, 78),
        line(20, 34, 42, 33), line(20, 44, 42, 43), line(20, 54, 40, 53),
        line(58, 33, 80, 34), line(58, 43, 80, 44),
    )


def calendar():
    dots = "".join(dot(x, y, 2.2) for x in (32, 50, 68) for y in (52, 64, 76))
    return j(
        rect(13, 22, 74, 66, 7),
        line(13, 41, 87, 41),
        line(32, 13, 32, 30), line(68, 13, 68, 30),
        dots,
    )


def clock():
    ticks = "".join(line(50 + 22 * math.cos(math.radians(a)), 50 + 22 * math.sin(math.radians(a)),
                         50 + 27 * math.cos(math.radians(a)), 50 + 27 * math.sin(math.radians(a)))
                    for a in range(0, 360, 90))
    return j(
        circle(50, 52, 34), line(50, 52, 50, 30), line(50, 52, 66, 60),
        rect(44, 6, 12, 8, 3), ticks,
    )


def coffee():
    return j(
        path("M24 42 L31 78 Q33 88 42 90 L58 90 Q67 88 69 78 L76 42 Z"),
        path("M76 50 C92 49 94 72 76 72"),
        ellipse(50, 91, 32, 5),
        path("M38 31 C33 25 41 21 36 13"),
        path("M52 31 C47 25 55 21 50 13"),
    )


def teapot():
    return j(
        path("M32 42 C20 48 20 76 36 82 C50 89 66 87 77 79 C87 71 84 50 72 44 Z"),
        path("M32 50 L14 44 L25 58 Z"),
        path("M77 53 C93 52 95 72 78 71"),
        rect(38, 30, 32, 9, 4),
        circle(54, 26, 4),
        path("M42 44 C48 40 62 40 68 44"),
    )


def pizza():
    return j(
        path("M50 10 L83 79 C69 89 57 86 50 82 C43 86 31 89 17 79 Z"),
        path("M17 79 C31 89 43 86 50 82 C57 86 69 89 83 79"),
        circle(48, 42, 5), circle(58, 60, 6), circle(36, 62, 5), circle(46, 73, 4),
    )


def burger():
    return j(
        path("M18 43 C18 23 33 13 50 13 C67 13 82 23 82 43 Z"),
        dot(38, 28, 2), dot(52, 24, 2), dot(65, 31, 2),
        path("M17 51 L25 45 L32 53 L39 45 L46 53 L53 45 L60 53 L67 45 L74 53 L83 47"),
        path("M19 60 Q50 55 81 60 Q50 67 19 60 Z"),
        path("M22 67 Q50 86 78 67 Z"),
    )


def icecream():
    return j(
        path("M35 47 L44 88 L56 88 L65 47 Z"),
        line(40, 56, 53, 76), line(60, 56, 47, 76), line(44, 64, 57, 64),
        circle(42, 41, 14), circle(58, 40, 15), circle(50, 27, 12),
        circle(50, 14, 4), path("M50 10 C53 5 59 5 62 8"),
    )


def donut():
    sprinkles = "".join(
        line(50 + 22 * math.cos(math.radians(a)), 50 + 22 * math.sin(math.radians(a)),
             50 + 28 * math.cos(math.radians(a + 12)), 50 + 28 * math.sin(math.radians(a + 12)))
        for a in range(0, 360, 40)
    )
    return j(circle(50, 50, 34), circle(50, 50, 13), sprinkles)


def cherry():
    return j(
        circle(36, 66, 12), circle(62, 70, 12),
        path("M36 54 C43 34 56 24 66 17"),
        path("M62 58 C64 42 66 30 66 17"),
        path("M66 17 C78 11 88 17 90 28 C79 32 69 27 66 17 Z"),
    )


def banana():
    return j(
        path("M20 32 C20 62 40 86 66 84 C82 83 89 68 84 55 C82 70 64 75 49 64 C36 54 34 42 37 32 Z"),
        line(37, 32, 30, 22),
    )


def apple():
    return j(
        path("M50 32 C37 19 17 28 19 48 C21 70 37 88 50 88 C63 88 79 70 81 48 C83 28 63 19 50 32 Z"),
        path("M50 32 C50 19 59 12 69 12 C69 23 60 32 50 32 Z"),
        line(50, 32, 50, 22),
    )


def camera():
    return j(
        rect(10, 34, 80, 46, 10),
        rect(34, 26, 24, 12, 4),
        circle(49, 57, 17), circle(49, 57, 8),
        circle(76, 45, 3), line(22, 34, 22, 28),
    )


def phone():
    return j(
        rect(30, 7, 40, 86, 10),
        rect(35, 18, 30, 63, 3),
        line(44, 14, 56, 14), line(44, 87, 56, 87),
    )


def laptop():
    return j(
        rect(20, 17, 60, 40, 4),
        rect(25, 22, 50, 30, 2),
        path("M10 63 L90 63 L80 78 L20 78 Z"),
        line(30, 68, 70, 68), line(30, 73, 62, 73),
    )


def headphones():
    return j(
        path("M19 60 C19 26 81 26 81 60"),
        rect(12, 54, 16, 26, 7),
        rect(72, 54, 16, 26, 7),
        line(20, 60, 20, 74), line(80, 60, 80, 74),
    )


def note():
    return j(
        ellipse(29, 72, 9, 7, rot=-20), ellipse(62, 64, 9, 7, rot=-20),
        line(37.5, 70, 37.5, 28), line(70.5, 62, 70.5, 20),
        path("M37.5 28 L70.5 20"), path("M37.5 38 L70.5 30"),
    )


def guitar():
    return j(
        circle(41, 64, 23), circle(41, 43, 17),
        rect(46, 8, 10, 30, 3),
        rect(43, 2, 16, 10, 3),
        circle(41, 55, 6),
        line(49, 20, 49, 78), line(52, 20, 52, 78),
    )


def mic():
    return j(
        rect(38, 8, 24, 44, 12),
        line(38, 22, 62, 22), line(38, 32, 62, 32), line(38, 42, 62, 42),
        path("M29 45 C29 67 71 67 71 45"),
        line(50, 67, 50, 86), line(36, 87, 64, 87),
    )


def gamepad():
    return j(
        path("M24 38 C13 38 8 53 13 66 C18 80 31 82 39 70 L45 63 L55 63 L61 70 C69 82 82 80 87 66 C92 53 87 38 76 38 Z"),
        path("M34 48 L34 60"), path("M28 54 L40 54"),
        dot(64, 49, 3.5), dot(73, 56, 3.5), dot(64, 63, 3.5),
    )


def ball():
    return j(
        circle(50, 50, 34),
        line(50, 16, 50, 84), line(17, 50, 83, 50),
        path("M28 24 C40 36 40 64 28 76"), path("M72 24 C60 36 60 64 72 76"),
    )


def cat():
    return j(
        circle(50, 54, 28),
        path("M26 42 L22 16 L42 31"), path("M74 42 L78 16 L58 31"),
        dot(40, 50, 3), dot(60, 50, 3),
        path("M46 60 Q50 64 54 60"),
        line(30, 60, 16, 56), line(30, 66, 16, 67),
        line(70, 60, 84, 56), line(70, 66, 84, 67),
    )


def dog():
    return j(
        circle(50, 52, 26),
        ellipse(25, 44, 10, 21, rot=18), ellipse(75, 44, 10, 21, rot=-18),
        dot(41, 48, 3), dot(59, 48, 3),
        ellipse(50, 63, 13, 9),
        circle(50, 60, 3.5),
        path("M50 64 L50 70"), path("M44 73 Q50 78 56 73"),
    )


def panda():
    return j(
        circle(50, 53, 28),
        circle(29, 28, 10), circle(71, 28, 10),
        ellipse(40, 51, 8, 11, rot=-12), ellipse(60, 51, 8, 11, rot=12),
        dot(40, 51, 2.6), dot(60, 51, 2.6),
        ellipse(50, 64, 5, 3.5),
        path("M45 70 Q50 75 55 70"),
    )


def fish():
    return j(
        path("M19 51 C30 29 62 27 78 51 C62 75 30 73 19 51 Z"),
        path("M78 51 L93 34 L93 68 Z"),
        path("M46 34 C51 42 51 60 46 68"),
        dot(32, 47, 3),
    )


def turtle():
    return j(
        path("M20 59 C20 34 35 22 50 22 C65 22 80 34 80 59 Z"),
        line(20, 59, 80, 59),
        path("M34 59 L34 43 L50 34 L66 43 L66 59"),
        circle(85, 61, 7),
        rect(26, 59, 8, 12, 3), rect(46, 59, 8, 12, 3), rect(66, 59, 8, 12, 3),
        path("M18 71 L26 64"),
    )


def bird():
    return j(
        path("M31 64 C18 53 22 32 40 28 C52 25 62 33 63 45 C64 57 55 66 43 66 Z"),
        circle(58, 44, 12),
        path("M69 42 L82 46 L69 49"),
        dot(59, 42, 2.4),
        path("M31 64 L18 76"),
        path("M40 42 C48 48 54 52 62 52"),
        line(43, 84, 43, 72), line(52, 84, 52, 73),
    )


def avatar():
    return j(
        circle(50, 35, 17),
        path("M16 88 C16 67 32 57 50 57 C68 57 84 67 84 88"),
    )


def robot():
    return j(
        rect(23, 24, 54, 46, 11),
        line(50, 24, 50, 10), circle(50, 7, 3),
        rect(15, 38, 8, 17, 2), rect(77, 38, 8, 17, 2),
        dot(38, 44, 3.5), dot(62, 44, 3.5),
        line(41, 58, 59, 58),
    )


def glasses():
    return j(
        circle(31, 52, 18), circle(69, 52, 18),
        path("M49 49 C53 44 47 44 51 49"),
        line(13, 50, 4, 42), line(87, 50, 96, 42),
    )


def umbrella():
    return j(
        path("M9 55 C9 28 32 10 50 10 C68 10 91 28 91 55 C81 46 70 46 62 55 C56 46 44 46 38 55 C30 46 19 46 9 55 Z"),
        line(50, 55, 50, 84),
        path("M50 84 C50 93 37 93 37 82"),
    )


def key():
    return j(
        circle(26, 50, 14),
        line(40, 50, 87, 50),
        line(70, 50, 70, 63), line(80, 50, 80, 60),
        line(38, 44, 38, 56),
    )


def bulb():
    return j(
        path("M50 10 C30 10 19 29 26 48 C30 59 38 62 38 73 L62 73 C62 62 70 59 74 48 C81 29 70 10 50 10 Z"),
        line(40, 79, 60, 79), line(42, 86, 58, 86),
        line(50, 55, 50, 44), dot(50, 40, 2.5),
    )


def gift():
    return j(
        rect(17, 42, 66, 42, 4),
        rect(13, 29, 74, 14, 4),
        line(50, 29, 50, 84),
        path("M50 29 C36 14 25 23 37 29"),
        path("M50 29 C64 14 75 23 63 29"),
    )


def balloon():
    return j(
        path("M50 9 C28 9 18 28 24 46 C29 61 42 66 50 76 C58 66 71 61 76 46 C82 28 72 9 50 9 Z"),
        path("M45 77 L55 77 L50 85 Z"),
        path("M50 85 C44 93 57 96 51 100"),
    )


def house():
    return j(
        path("M10 47 L50 12 L90 47"),
        rect(19, 47, 62, 41),
        rect(44, 63, 16, 25, 2),
        rect(28, 55, 13, 11, 1),
        line(60, 56, 74, 56), line(60, 61, 74, 61),
    )


def car():
    return j(
        path("M9 63 L16 47 C19 39 28 35 40 35 L61 35 C73 35 81 41 85 49 L89 63 Z"),
        circle(29, 71, 10), circle(70, 71, 10),
        path("M22 47 L34 35"), path("M46 35 L46 47"), path("M58 35 L58 47"), path("M70 37 L78 47"),
        line(9, 55, 89, 55),
    )


def lipstick():
    return j(
        rect(37, 49, 26, 34, 3),
        path("M37 49 L37 22 C37 13 63 13 63 22 L63 49 Z"),
        line(37, 58, 63, 58),
        path("M43 22 L43 40"), path("M50 19 L50 38"),
    )


def trophy():
    return j(
        path("M30 17 L70 17 L66 45 C64 59 56 67 50 67 C44 67 36 59 34 45 Z"),
        path("M30 24 C16 24 16 45 33 47"),
        path("M70 24 C84 24 84 45 67 47"),
        line(50, 67, 50, 79),
        rect(34, 79, 32, 8, 2),
    )


def crown():
    return j(
        path("M18 68 L14 25 L34 43 L50 17 L66 43 L86 25 L82 68 Z"),
        line(18, 59, 82, 59),
        dot(50, 51, 3), dot(34, 51, 2.5), dot(66, 51, 2.5),
    )


def ghost():
    return j(
        path("M23 86 L23 48 C23 26 36 13 50 13 C64 13 77 26 77 48 L77 86 L66 77 L58 86 L50 77 L42 86 L34 77 Z"),
        dot(40, 46, 2.6), dot(60, 46, 2.6),
        ellipse(50, 59, 6, 4),
    )


def battery():
    return j(
        rect(16, 35, 62, 32, 5),
        rect(78, 45, 8, 12, 2),
        path("M49 40 L39 53 L50 53 L43 65 L58 50 L47 50 Z"),
    )


def plug():
    return j(
        rect(30, 29, 40, 35, 8),
        line(40, 29, 40, 13), line(60, 29, 60, 13),
        path("M50 64 C50 82 34 80 34 94"),
    )


def pin():
    return j(
        path("M50 91 C50 91 22 58 22 40 C22 24 34 11 50 11 C66 11 78 24 78 40 C78 58 50 91 50 91 Z"),
        circle(50, 40, 8),
    )


def dice():
    dots = j(dot(36, 36, 2.6), dot(64, 36, 2.6), dot(50, 50, 2.6), dot(36, 64, 2.6), dot(64, 64, 2.6))
    return j(rect(21, 21, 58, 58, 11), dots)


def cassette():
    return j(
        rect(12, 31, 76, 42, 6),
        rect(24, 39, 52, 17, 2),
        circle(35, 48, 6), circle(65, 48, 6),
        circle(38, 64, 4), circle(62, 64, 4),
        line(44, 64, 56, 64),
    )


def tv():
    return j(
        rect(14, 24, 72, 48, 6),
        line(34, 24, 24, 8), line(66, 24, 76, 8),
        line(38, 81, 62, 81), line(50, 72, 50, 81),
        rect(23, 32, 54, 31, 2),
    )


def cupcake():
    return j(
        path("M27 51 L35 88 L65 88 L73 51 Z"),
        line(36, 57, 39, 82), line(47, 57, 48, 82), line(58, 57, 57, 82), line(69, 57, 65, 82),
        path("M26 50 C19 43 25 34 33 36 C35 25 47 21 53 28 C60 19 74 24 72 36 C83 37 85 49 76 52 Z"),
        circle(50, 18, 4),
    )


def mushroom():
    return j(
        path("M13 52 C13 30 30 15 50 15 C70 15 87 30 87 52 Z"),
        path("M36 52 C36 68 34 82 40 89 L60 89 C66 82 64 68 64 52"),
        circle(34, 37, 3.5), circle(51, 28, 3), circle(68, 39, 3.5),
    )


def cactus():
    return j(
        path("M40 87 L40 41 C40 30 60 30 60 41 L60 87"),
        path("M40 56 C27 56 25 43 25 36"), path("M25 36 C25 29 33 29 33 36 L33 47"),
        path("M60 64 C73 64 75 51 75 44"), path("M75 44 C75 37 67 37 67 44 L67 55"),
        path("M32 87 L68 87"),
    )


def strawberry():
    return j(
        path("M50 88 C28 76 20 58 25 42 C33 24 67 24 75 42 C80 58 72 76 50 88 Z"),
        path("M29 35 L22 24 L38 29 L50 18 L62 29 L78 24 L71 35"),
        dot(38, 50, 2), dot(50, 48, 2), dot(62, 50, 2),
        dot(44, 62, 2), dot(56, 62, 2), dot(50, 74, 2),
    )


def paint():
    return j(
        path("M50 14 C27 14 10 30 10 51 C10 72 27 87 47 87 C56 87 58 80 53 76 C48 72 53 65 60 65 L75 65 C85 65 91 57 90 48 C89 28 71 14 50 14 Z"),
        dot(32, 34, 4), dot(50, 27, 4), dot(68, 35, 4), dot(28, 50, 4),
    )


def scissors():
    return j(
        circle(28, 75, 10), circle(50, 75, 10),
        line(33, 67, 80, 15), line(45, 67, 70, 20),
        circle(36, 62, 2), circle(47, 57, 2),
    )


def eye():
    return j(
        path("M10 51 C24 31 42 24 50 24 C58 24 76 31 90 51 C76 71 58 78 50 78 C42 78 24 71 10 51 Z"),
        circle(50, 51, 12), circle(50, 51, 4),
        line(28, 31, 24, 23), line(51, 24, 51, 15), line(72, 31, 76, 23),
    )


def alien():
    return j(
        path("M50 13 C31 13 20 31 24 47 C27 60 36 65 38 79 L62 79 C64 65 73 60 76 47 C80 31 69 13 50 13 Z"),
        ellipse(40, 43, 7, 10), ellipse(60, 43, 7, 10),
        dot(40, 43, 2.4), dot(60, 43, 2.4),
        path("M36 21 C31 10 24 8 19 11"), dot(18, 10, 2.5),
        path("M64 21 C69 10 76 8 81 11"), dot(82, 10, 2.5),
        path("M44 67 Q50 72 56 67"),
    )


def peace():
    return j(
        rect(35, 48, 30, 34, 9),
        rect(37, 20, 9, 34, 4), rect(54, 16, 9, 38, 4),
        path("M37 30 C30 32 28 40 33 44"),
        path("M35 48 L33 39"), path("M65 48 L67 40"),
        line(40, 82, 60, 82),
    )


def lollipop():
    return j(
        circle(50, 35, 23),
        path("M50 22 C57 22 63 28 63 35 C63 42 57 48 50 48 C43 48 37 42 37 35 C37 29 42 24 48 24"),
        line(52, 57, 56, 88),
        path("M46 24 C52 16 62 16 68 22"),
    )


def anchor():
    return j(
        circle(50, 18, 8),
        line(50, 26, 50, 82),
        line(34, 40, 66, 40),
        path("M20 58 C22 78 36 88 50 88 C64 88 78 78 80 58"),
        path("M20 58 L12 50"), path("M20 58 L28 50"),
        path("M80 58 L72 50"), path("M80 58 L88 50"),
    )


def hanger():
    return j(
        path("M50 16 C38 16 34 28 42 34 C46 37 50 39 50 45"),
        path("M50 45 L14 72 C12 74 14 78 18 78 L82 78 C86 78 88 74 86 72 Z"),
    )


def globe():
    return j(
        circle(50, 46, 28),
        ellipse(50, 46, 13, 28),
        line(23, 38, 77, 38), line(22, 54, 78, 54),
        path("M30 82 C38 74 62 74 70 82"),
        line(50, 74, 50, 86),
    )


MOTIFS = {
    "chat": chat,
    "chat_heart": chat_heart,
    "heart": heart,
    "star": star5,
    "smile": smile,
    "wink": wink,
    "flower": flower,
    "leaf": leaf,
    "cloud": cloud,
    "sun": sun,
    "moon": moon,
    "planet": planet,
    "rocket": rocket,
    "plane": plane,
    "envelope": envelope,
    "pencil": pencil,
    "book": book,
    "calendar": calendar,
    "clock": clock,
    "coffee": coffee,
    "teapot": teapot,
    "pizza": pizza,
    "burger": burger,
    "icecream": icecream,
    "donut": donut,
    "cherry": cherry,
    "banana": banana,
    "apple": apple,
    "camera": camera,
    "phone": phone,
    "laptop": laptop,
    "headphones": headphones,
    "note": note,
    "guitar": guitar,
    "mic": mic,
    "gamepad": gamepad,
    "ball": ball,
    "cat": cat,
    "dog": dog,
    "panda": panda,
    "fish": fish,
    "turtle": turtle,
    "bird": bird,
    "avatar": avatar,
    "robot": robot,
    "glasses": glasses,
    "umbrella": umbrella,
    "key": key,
    "bulb": bulb,
    "gift": gift,
    "balloon": balloon,
    "house": house,
    "car": car,
    "lipstick": lipstick,
    "trophy": trophy,
    "crown": crown,
    "ghost": ghost,
    "battery": battery,
    "plug": plug,
    "pin": pin,
    "dice": dice,
    "cassette": cassette,
    "tv": tv,
    "cupcake": cupcake,
    "mushroom": mushroom,
    "cactus": cactus,
    "strawberry": strawberry,
    "paint": paint,
    "scissors": scissors,
    "eye": eye,
    "alien": alien,
    "peace": peace,
    "lollipop": lollipop,
    "anchor": anchor,
    "hanger": hanger,
    "globe": globe,
}


# ---------------------------------------------------------------- tiny fillers

def tiny_dot():
    return j(circle(50, 50, 13), dot(50, 50, 2.5))


def tiny_ring():
    return circle(50, 50, 9)


def tiny_star():
    return star(50, 50, 22, 9)


def tiny_heart():
    return path("M50 78 C30 64 24 51 31 41 C38 31 48 35 50 43 C52 35 62 31 69 41 C76 51 70 64 50 78 Z")


def tiny_sparkle():
    return path("M50 12 C53 36 64 47 88 50 C64 53 53 64 50 88 C47 64 36 53 12 50 C36 47 47 36 50 12 Z")


def tiny_flower():
    petals = "".join(circle(50 + 9 * math.cos(math.radians(a)), 50 + 9 * math.sin(math.radians(a)), 5) for a in range(0, 360, 72))
    return j(petals, circle(50, 50, 3.5))


def tiny_leaf():
    return j(path("M50 12 C70 30 72 58 50 88 C28 58 30 30 50 12 Z"), path("M50 24 L50 76"))


def tiny_squiggle():
    return path("M14 55 C28 34 40 76 54 54 C64 38 74 46 86 55")


def tiny_note():
    return j(ellipse(38, 68, 7, 5, rot=-18), line(44, 66, 44, 33), path("M44 33 L68 27"), line(68, 27, 68, 56), ellipse(62, 59, 7, 5, rot=-18))


def tiny_moon():
    return path("M62 18 A30 30 0 1 0 62 82 A24 24 0 1 1 62 18 Z")


def tiny_cloud():
    return path("M24 66 C16 64 16 54 24 52 C24 42 38 38 44 45 C50 36 66 38 68 49 C78 50 80 64 70 67 Z")


def tiny_plus():
    return j(line(50, 32, 50, 68), line(32, 50, 68, 50))


def tiny_pebble():
    return circle(50, 50, 15, fill="currentColor", stroke="none")


def tiny_arrow():
    return j(path("M16 70 C36 42 56 38 82 30"), path("M82 30 L68 26"), path("M82 30 L74 42"))


TINY = {
    "dot": tiny_dot,
    "ring": tiny_ring,
    "star": tiny_star,
    "heart": tiny_heart,
    "sparkle": tiny_sparkle,
    "flower": tiny_flower,
    "leaf": tiny_leaf,
    "squiggle": tiny_squiggle,
    "note": tiny_note,
    "moon": tiny_moon,
    "cloud": tiny_cloud,
    "arrow": tiny_arrow,
    "plus": tiny_plus,
    "pebble": tiny_pebble,
}
