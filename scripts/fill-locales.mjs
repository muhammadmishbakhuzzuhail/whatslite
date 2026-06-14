// fill-locales.mjs — isi HANYA kunci yang HILANG di tiap locale mesin dengan
// machine-translate (Google gtx) dari en.json. Terjemahan lama TIDAK disentuh
// (incremental, hemat: cuma kunci baru). Jalankan tiap kali menambah kunci ke
// en.json agar 70 locale tak ketinggalan (sebelumnya hanya en-fallback).
//
// Run:  node scripts/fill-locales.mjs
//
// Sumber kebenaran = en.json. id/en/es hand-authored → dilewati.
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { TRANSLATE_LANGS } from "../frontend/src/lib/langs.js";

const __dir = path.dirname(fileURLToPath(import.meta.url));
const LDIR = path.join(__dir, "../frontend/src/lib/locales");
const SKIP = new Set(["id", "en", "es"]);
const en = JSON.parse(fs.readFileSync(path.join(LDIR, "en.json"), "utf8"));

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const PH = /\{[^}]+\}|%[a-zA-Z]/g;
function protect(s) { const toks = []; const masked = s.replace(PH, (m) => { toks.push(m); return `[[${toks.length - 1}]]`; }); return { masked, toks }; }
function restore(s, toks) { return s.replace(/\[\[(\d+)\]\]/g, (_, i) => toks[+i] ?? ""); }

async function gtx(text, tl) {
  const u = "https://translate.googleapis.com/translate_a/single?client=gtx&sl=en&tl=" +
    encodeURIComponent(tl) + "&dt=t&q=" + encodeURIComponent(text);
  for (let a = 0; a < 4; a++) {
    try { const r = await fetch(u, { headers: { "User-Agent": "Mozilla/5.0" } }); if (r.status === 200) { const j = await r.json(); return j[0].map((s) => s[0]).join(""); } } catch (e) {}
    await sleep(500 * (a + 1));
  }
  return null;
}

async function fill(code, file, dict, missing) {
  const masks = missing.map((k) => protect(en[k]));
  const CHUNK = 70;
  for (let i = 0; i < missing.length; i += CHUNK) {
    const idx = missing.slice(i, i + CHUNK);
    const src = masks.slice(i, i + CHUNK);
    const res = await gtx(src.map((m) => m.masked).join("\n"), code);
    const lines = res ? res.split("\n") : null;
    idx.forEach((k, j) => { dict[k] = (lines && lines.length === idx.length) ? restore(lines[j], src[j].toks) : en[k]; });
    await sleep(150);
  }
  fs.writeFileSync(file, JSON.stringify(dict, null, 1));
}

const main = async () => {
  const enKeys = Object.keys(en);
  let touched = 0;
  for (const { code } of TRANSLATE_LANGS) {
    if (SKIP.has(code)) continue;
    const file = path.join(LDIR, code + ".json");
    if (!fs.existsSync(file)) { console.log("no file, skip", code, "(run gen-locales.mjs)"); continue; }
    const dict = JSON.parse(fs.readFileSync(file, "utf8"));
    const missing = enKeys.filter((k) => !(k in dict));
    if (missing.length === 0) { continue; }
    process.stdout.write(`fill ${code} (+${missing.length}) … `);
    await fill(code, file, dict, missing);
    console.log("ok");
    touched++;
  }
  console.log(`done — ${touched} locales updated`);
};
main();
