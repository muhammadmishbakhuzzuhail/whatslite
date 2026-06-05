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
  // 🌿 Flora & Fauna — daun, tunas, bunga, monstera + jejak kaki, ikan, burung, kupu.
  flora: [
    "M18 3 C7 11 7 27 18 33 C29 27 29 11 18 3 Z M18 7 V31",
    "M18 33 V18 M18 22 C11 21 8 13 14 11 C17 16 18 19 18 22 M18 20 C25 19 28 11 22 9 C19 14 18 17 18 20",
    "M18 16 a2.5 2.5 0 1 0 0.1 0 M18 9 a3 3 0 1 0 0.1 0 M18 23 a3 3 0 1 0 0.1 0 M11 16 a3 3 0 1 0 0.1 0 M25 16 a3 3 0 1 0 0.1 0",
    "M8 29 C4 17 12 6 24 8 C30 9 32 16 30 22 C28 28 20 31 8 29 Z M16 12 L18 20 M24 12 L22 22 M12 20 L20 24",
    "M12 16 a2 2.5 0 1 0 0.1 0 M18 14 a2 2.5 0 1 0 0.1 0 M24 16 a2 2.5 0 1 0 0.1 0 M18 21 a4 3.5 0 1 0 0.1 0",
    "M6 18 C12 10 22 10 27 18 C22 26 12 26 6 18 Z M27 14 L32 11 V25 L27 22 M11 16 a1 1 0 1 0 0.1 0",
    "M8 20 C10 14 16 12 20 14 C24 10 30 12 30 12 C28 16 26 17 24 17 C26 20 22 24 16 22 C12 26 8 24 8 20 Z M28 14 a0.8 0.8 0 1 0 0.1 0",
    "M18 8 V28 M18 12 C12 6 6 8 8 14 C5 16 6 22 12 22 C16 22 18 18 18 16 M18 12 C24 6 30 8 28 14 C31 16 30 22 24 22 C20 22 18 18 18 16",
  ],
  // 🌙 Angkasa — bintang, bulan, planet, roket, matahari, komet, rasi.
  angkasa: [
    "M18 4 L21 15 L32 18 L21 21 L18 32 L15 21 L4 18 L15 15 Z",
    "M23 6 A13 13 0 1 0 23 30 A10 10 0 1 1 23 6 Z",
    "M18 18 m-7 0 a7 7 0 1 0 14 0 a7 7 0 1 0 -14 0 M7 20 C13 24 23 16 29 14",
    "M18 4 C24 9 24 18 21 24 H15 C12 18 12 9 18 4 Z M15 24 L12 30 L16 27 M21 24 L24 30 L20 27 M16 13 a2 2 0 1 0 4 0 a2 2 0 1 0 -4 0",
    "M18 13 a5 5 0 1 0 0.1 0 M18 5 V9 M18 27 V31 M5 18 H9 M27 18 H31 M9 9 L12 12 M24 24 L27 27 M27 9 L24 12 M9 27 L12 24",
    "M24 12 a4 4 0 1 0 0.1 0 M21 15 L8 28 M24 18 L13 29 M18 13 L6 25",
    "M18 8 L19.5 16.5 L28 18 L19.5 19.5 L18 28 L16.5 19.5 L8 18 L16.5 16.5 Z",
    "M8 10 L16 18 L14 27 M16 18 L26 14 L30 24 M8 10 a1 1 0 1 0 0.1 0 M16 18 a1 1 0 1 0 0.1 0 M14 27 a1 1 0 1 0 0.1 0 M26 14 a1 1 0 1 0 0.1 0 M30 24 a1 1 0 1 0 0.1 0",
  ],
  // 💻 Teknologi — kamera, laptop, ponsel, headphone, chip, lampu, gamepad, wifi.
  teknologi: [
    "M6 12 H11 L13 9 H23 L25 12 H30 V28 H6 Z M18 14 a6 6 0 1 0 0.1 0 Z",
    "M9 11 H27 V23 H9 Z M5 27 H31 L29 23 H7 Z M12 14 H24 V20 H12 Z",
    "M12 5 H24 V31 H12 Z M16 8 H20 M16 28 a2 2 0 1 0 4 0 a2 2 0 1 0 -4 0",
    "M8 20 V17 a10 10 0 0 1 20 0 V20 M6 20 H10 V27 H6 Z M26 20 H30 V27 H26 Z",
    "M11 11 H25 V25 H11 Z M7 15 H11 M7 21 H11 M25 15 H29 M25 21 H29 M15 7 V11 M21 7 V11 M15 25 V29 M21 25 V29 M16 16 H20 V20 H16 Z",
    "M18 4 a8 8 0 0 1 5 14 C21 20 21 22 21 24 H15 C15 22 15 20 13 18 A8 8 0 0 1 18 4 Z M15 27 H21 M16 30 H20",
    "M11 14 H25 C29 14 31 24 27 25 C24 26 23 21 20 21 H16 C13 21 12 26 9 25 C5 24 7 14 11 14 Z M13 17 V20 M11.5 18.5 H14.5 M22 17 a1 1 0 1 0 0.1 0 M25 19 a1 1 0 1 0 0.1 0",
    "M6 16 C12 9 24 9 30 16 M10 20 C15 15 21 15 26 20 M14 24 C16 22 20 22 22 24 M17.5 27.5 H18.5",
  ],
};

// Sebaran 3×3 dgn jitter + rotasi/skala variatif (organik tapi deterministik).
const SPOTS = [
  [40, 46, 1.0, 12], [150, 30, 0.85, -20], [262, 52, 1.1, 30],
  [60, 150, 0.9, -8], [170, 140, 1.15, 18], [286, 158, 0.8, -26],
  [36, 250, 1.05, 22], [150, 262, 0.92, -14], [268, 256, 1.0, 8],
];

function buildSVG(cat, stroke) {
  const gl = GLYPHS[cat] || GLYPHS.aneka;
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
