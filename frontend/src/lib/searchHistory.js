// searchHistory.js — riwayat & autocomplete pencarian GIF/stiker (localStorage).
// kind = "gif" | "sticker".
const KEY = (k) => "wa-hist-" + k;
const MAX = 14;

// Kata kunci populer (seed autocomplete saat riwayat kosong).
export const POPULAR = [
  "lol", "love", "sad", "happy", "wow", "ok", "thanks", "hi", "bye", "angry",
  "dance", "clap", "cry", "kiss", "hug", "yes", "no", "please", "sorry", "congrats",
  "facepalm", "thumbs up", "good night", "good morning", "miss you", "wait", "shocked", "laugh",
];

export function getHistory(kind) {
  try {
    const v = JSON.parse(localStorage.getItem(KEY(kind)) || "[]");
    return Array.isArray(v) ? v : [];
  } catch (e) { return []; }
}

export function addHistory(kind, term) {
  term = (term || "").trim();
  if (!term) return getHistory(kind);
  let h = getHistory(kind).filter((x) => x.toLowerCase() !== term.toLowerCase());
  h = [term, ...h].slice(0, MAX);
  try { localStorage.setItem(KEY(kind), JSON.stringify(h)); } catch (e) {}
  return h;
}

export function removeHistory(kind, term) {
  const h = getHistory(kind).filter((x) => x.toLowerCase() !== (term || "").toLowerCase());
  try { localStorage.setItem(KEY(kind), JSON.stringify(h)); } catch (e) {}
  return h;
}

export function clearHistory(kind) {
  try { localStorage.removeItem(KEY(kind)); } catch (e) {}
  return [];
}

// Saran autocomplete utk query: cocokkan riwayat lalu populer (substring), unik.
export function suggest(query, hist) {
  const s = (query || "").trim().toLowerCase();
  if (!s) return [];
  const out = [], seen = new Set();
  for (const t of [...(hist || []), ...POPULAR]) {
    const k = t.toLowerCase();
    if (k !== s && k.includes(s) && !seen.has(k)) { seen.add(k); out.push(t); }
    if (out.length >= 8) break;
  }
  return out;
}
