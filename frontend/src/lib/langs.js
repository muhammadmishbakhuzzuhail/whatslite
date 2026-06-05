// Daftar bahasa target terjemahan (kode ISO Google Translate). Sumber dideteksi
// otomatis di BE, jadi ini hanya bahasa TUJUAN. ~tak perlu install apa pun (online).
export const TRANSLATE_LANGS = [
  { code: "id", name: "Indonesia" },
  { code: "en", name: "English" },
  { code: "ms", name: "Melayu" },
  { code: "ar", name: "العربية" },
  { code: "zh-CN", name: "中文 (简体)" },
  { code: "ja", name: "日本語" },
  { code: "ko", name: "한국어" },
  { code: "es", name: "Español" },
  { code: "fr", name: "Français" },
  { code: "de", name: "Deutsch" },
  { code: "ru", name: "Русский" },
  { code: "pt", name: "Português" },
  { code: "it", name: "Italiano" },
  { code: "nl", name: "Nederlands" },
  { code: "tr", name: "Türkçe" },
  { code: "vi", name: "Tiếng Việt" },
  { code: "th", name: "ไทย" },
  { code: "hi", name: "हिन्दी" },
  { code: "bn", name: "বাংলা" },
  { code: "ta", name: "தமிழ்" },
  { code: "ur", name: "اردو" },
  { code: "fa", name: "فارسی" },
  { code: "fil", name: "Filipino" },
  { code: "pl", name: "Polski" },
  { code: "uk", name: "Українська" },
];

export function langName(code) {
  const l = TRANSLATE_LANGS.find((x) => x.code === code);
  return l ? l.name : code;
}
