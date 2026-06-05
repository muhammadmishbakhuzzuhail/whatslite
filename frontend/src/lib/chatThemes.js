// chatThemes.js — tema latar chat (kurasi). Default = doodle WhatsApp generik;
// Polos = warna saja; sisanya = doodle BERTEMA kategori (lihat lib/doodles.js):
// pola line-art ikon kategori, opacity rendah, di atas warna latar tema.
// Tiap tema punya warna light & dark agar enak di kedua mode app.
//
// Dipakai stores.applyChatTheme() → set CSS var --chat-bg-* & --chat-doodle.

export const CHAT_THEMES = [
  { id: "default",   label: "Default",        cat: "aneka",     light: "#eef1f6", dark: "#0a0f14" },
  { id: "plain",     label: "Polos",          doodle: false,    light: "#eef1f6", dark: "#0a0f14" },
  { id: "flora",     label: "Flora & Fauna",  cat: "flora",     light: "#eef3ec", dark: "#0c140f" },
  { id: "angkasa",   label: "Angkasa",        cat: "angkasa",   light: "#eceef6", dark: "#0a0d1a" },
  { id: "teknologi", label: "Teknologi",      cat: "teknologi", light: "#e9eef4", dark: "#0b1016" },
];

export function chatThemeById(id) {
  return CHAT_THEMES.find((t) => t.id === id) || CHAT_THEMES[0];
}

// Warna swatch utk pratinjau tombol (sesuai mode aktif).
export function chatThemeSwatch(id, dark) {
  const t = chatThemeById(id);
  return dark ? t.dark : t.light;
}
