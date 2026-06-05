// doodles.js — pola latar chat bertema kategori (line-art tiled, opacity rendah).
// Tiap kategori = kumpulan glyph (path digambar dlm kotak ~0..36). buildSVG
// menyebar glyph di tile 330×330 (posisi/rotasi/skala tetap → deterministik,
// hasil sama tiap render) lalu di-encode jadi data-URI utk --chat-doodle.
// Warna stroke beda utk light vs dark → 1 set glyph, 2 nada.

const GLYPHS = {
  // 🗂️ Aneka — generik sehari-hari/komunikasi (pengganti PNG Default agar
  // opacity seragam dgn kategori): gelembung chat, hati, hadiah, bintang,
  // not balok, kamera, daun, plus.
  aneka: [
    "M6 10 H30 V24 H16 L10 30 V24 H6 Z",
    "M18 30 C6 22 6 10 13 10 C16 10 18 13 18 14 C18 13 20 10 23 10 C30 10 30 22 18 30 Z",
    "M7 16 H29 V30 H7 Z M7 16 V13 H29 V16 M18 13 V30 M18 13 C16 6 9 8 12 12 C14 13 16 13 18 13 M18 13 C20 6 27 8 24 12 C22 13 20 13 18 13",
    "M18 4 L21 15 L32 18 L21 21 L18 32 L15 21 L4 18 L15 15 Z",
    "M14 27 a4 3 0 1 0 8 0 a4 3 0 1 0 -8 0 M22 27 V8 L28 11 V13 M14 27 V14 L22 11",
    "M6 12 H11 L13 9 H23 L25 12 H30 V28 H6 Z M18 14 a6 6 0 1 0 0.1 0 Z",
    "M18 3 C9 9 9 23 18 29 C27 23 27 9 18 3 Z M18 7 V27",
    "M18 11 V25 M11 18 H25",
  ],
  // 🌿 Botani — daun, tunas, frond, monstera.
  botani: [
    "M18 3 C7 11 7 27 18 33 C29 27 29 11 18 3 Z M18 7 V31",
    "M18 33 V18 M18 22 C11 21 8 13 14 11 C17 16 18 19 18 22 M18 20 C25 19 28 11 22 9 C19 14 18 17 18 20",
    "M18 34 V6 M18 12 L11 8 M18 12 L25 8 M18 18 L10 14 M18 18 L26 14 M18 24 L11 21 M18 24 L25 21",
    "M8 29 C4 17 12 6 24 8 C30 9 32 16 30 22 C28 28 20 31 8 29 Z M16 12 L18 20 M24 12 L22 22 M12 20 L20 24",
  ],
  // ⚡ Sirkuit — chip, wifi, resistor, bolt, node.
  sirkuit: [
    "M11 11 H25 V25 H11 Z M7 15 H11 M7 21 H11 M25 15 H29 M25 21 H29 M15 7 V11 M21 7 V11 M15 25 V29 M21 25 V29 M16 16 H20 V20 H16 Z",
    "M6 16 C12 9 24 9 30 16 M10 20 C15 15 21 15 26 20 M14 24 C16 22 20 22 22 24 M17.5 27.5 H18.5",
    "M5 18 H12 L14 12 L18 24 L22 12 L26 24 L28 18 H33",
    "M20 5 L12 21 H19 L16 33 L26 16 H19 Z",
    "M18 8 V14 M18 28 V22 M8 18 H14 M28 18 H22 M14 18 a4 4 0 1 0 8 0 a4 4 0 1 0 -8 0",
  ],
  // 🌙 Angkasa — bintang, bulan, planet, roket.
  angkasa: [
    "M18 4 L21 15 L32 18 L21 21 L18 32 L15 21 L4 18 L15 15 Z",
    "M23 6 A13 13 0 1 0 23 30 A10 10 0 1 1 23 6 Z",
    "M9 18 a9 9 0 1 0 18 0 a9 9 0 1 0 -18 0 M6 22 C12 26 24 26 30 22",
    "M18 4 C24 9 24 18 21 24 H15 C12 18 12 9 18 4 Z M15 24 L12 30 L16 27 M21 24 L24 30 L20 27 M16 13 a2 2 0 1 0 4 0 a2 2 0 1 0 -4 0",
    "M18 13 V23 M13 18 H23",
  ],
  // ◇ Geometris — lingkaran, segitiga, plus, gelombang, konsentris, wajik.
  geometris: [
    "M8 18 a10 10 0 1 0 20 0 a10 10 0 1 0 -20 0",
    "M18 6 L31 29 H5 Z",
    "M18 8 V28 M8 18 H28",
    "M5 18 Q11 10 18 18 T31 18",
    "M13 18 a5 5 0 1 0 10 0 a5 5 0 1 0 -10 0 M8 18 a10 10 0 1 0 20 0 a10 10 0 1 0 -20 0",
    "M18 5 L30 18 L18 31 L6 18 Z",
  ],
  // 🍉 Tropis — semangka, nanas, ceri, jeruk.
  tropis: [
    "M6 12 C14 28 22 28 30 12 C22 18 14 18 6 12 Z M18 14 V22 M14 14 L13 20 M22 14 L23 20",
    "M14 14 C12 16 12 22 14 26 C16 30 20 30 22 26 C24 22 24 16 22 14 M18 14 V26 M15 18 H21 M15 22 H21 M18 14 L14 6 M18 13 V5 M18 14 L22 6",
    "M11 22 a4 4 0 1 0 8 0 a4 4 0 1 0 -8 0 M21 25 a3.5 3.5 0 1 0 7 0 a3.5 3.5 0 1 0 -7 0 M15 18 C17 10 23 10 24.5 21.5 M15 18 C19 15 23 17 24 21",
    "M8 18 a10 10 0 1 0 20 0 a10 10 0 1 0 -20 0 M18 8 V28 M8 18 H28 M11 11 L25 25 M25 11 L11 25",
  ],
};

// Sebaran 3×3 dgn jitter + rotasi/skala variatif (organik tapi deterministik).
const SPOTS = [
  [40, 46, 1.0, 12], [150, 30, 0.85, -20], [262, 52, 1.1, 30],
  [60, 150, 0.9, -8], [170, 140, 1.15, 18], [286, 158, 0.8, -26],
  [36, 250, 1.05, 22], [150, 262, 0.92, -14], [268, 256, 1.0, 8],
];

function buildSVG(cat, stroke) {
  const gl = GLYPHS[cat] || GLYPHS.geometris;
  let inner = "";
  SPOTS.forEach((p, i) => {
    inner += `<g transform="translate(${p[0]} ${p[1]}) scale(${p[2]}) rotate(${p[3]} 18 18)"><path d="${gl[i % gl.length]}"/></g>`;
  });
  const svg = `<svg xmlns='http://www.w3.org/2000/svg' width='330' height='330' viewBox='0 0 330 330' fill='none' stroke='${stroke}' stroke-width='1.7' stroke-linecap='round' stroke-linejoin='round'>${inner}</svg>`;
  return "data:image/svg+xml," + encodeURIComponent(svg);
}

// URL doodle utk satu kategori sesuai nada (dark → garis putih samar).
// alpha: opacity garis. Default sangat samar (latar chat). Swatch pakai lebih
// tinggi agar pratinjau tetap kebaca di kotak kecil.
export function doodleURI(cat, dark, alpha) {
  const a = alpha != null ? alpha : (dark ? 0.04 : 0.05);
  return buildSVG(cat, dark ? `rgba(255,255,255,${a})` : `rgba(15,23,32,${a})`);
}

export const DOODLE_CATS = Object.keys(GLYPHS);
