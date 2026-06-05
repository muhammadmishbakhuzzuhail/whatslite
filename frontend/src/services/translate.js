// ============================================================
// services/translate.js — SEAM translate PESAN dinamis (beda dari i18n statis).
//
// MESIN PRODUKSI (offline, privat — pilihan utama untuk pesan E2E):
//   ► Bergamot (Firefox Translations, Mozilla) — NMT on-device via WASM.
//     Privat (tak ada data keluar), model per-pasangan-bahasa (~15–40MB).
//   Alternatif: Argos Translate, atau LibreTranslate self-host.
//   Idealnya dijalankan di engine Go: window.go.main.App.Translate(text, target).
//
// SEKARANG: translator DEMO (peta frasa) agar fitur & UI terlihat bekerja
//   sebelum BE + model Bergamot dipasang. Ganti isi translateMessage() saat itu.
// ============================================================

const demo = {
  en: {
    "Halo! Jadi nanti malam ngumpul jam berapa?": "Hi! So what time are we meeting tonight?",
    "Oke sip. Oh iya aku bawa kamera, sekalian foto-foto. Kamu bawa speaker yang kemarin gak?":
      "Okay cool. Oh, I'll bring the camera to take photos too. Are you bringing the speaker from yesterday?",
    "Spot kemarin, bagus banget buat sunset 🌅": "Yesterday's spot, perfect for the sunset 🌅",
    "Nanti malam kita makan di luar ya 🍽️": "Let's eat out tonight 🍽️",
    "Setuju! Mau makan apa?": "Agreed! What do you want to eat?",
    "Sushi atau Padang?": "Sushi or Padang food?",
    "Tempatnya di sini ya 📍": "The place is right here 📍",
    "Jangan lupa makan ya nak": "Don't forget to eat, sweetie",
    "Hmm belum pasti nih, masih nunggu kabar": "Hmm, not sure yet, still waiting to hear back",
    "Oke besok aku kabarin lagi": "Okay, I'll let you know again tomorrow",
  },
  es: {
    "Halo! Jadi nanti malam ngumpul jam berapa?": "¡Hola! ¿A qué hora nos juntamos esta noche?",
    "Spot kemarin, bagus banget buat sunset 🌅": "El lugar de ayer, perfecto para el atardecer 🌅",
    "Nanti malam kita makan di luar ya 🍽️": "Salgamos a comer esta noche 🍽️",
    "Setuju! Mau makan apa?": "¡De acuerdo! ¿Qué quieres comer?",
    "Sushi atau Padang?": "¿Sushi o comida Padang?",
    "Tempatnya di sini ya 📍": "El lugar está aquí 📍",
    "Jangan lupa makan ya nak": "No olvides comer, cariño",
  },
};

const A = typeof window !== "undefined" ? window.go?.app?.App : null;

// App native → mesin terjemah nyata (Google via engine Go).
// Browser/preview → stub demo (peta frasa) agar UI tetap terlihat bekerja.
export async function translateMessage(text, target = "en") {
  if (A && A.Translate) {
    try {
      const out = await A.Translate(text, target);
      if (out) return out;
    } catch (e) {}
  }
  const table = demo[target] || {};
  return table[text] || (target === "id" ? text : `(${target}) ${text}`);
}

export const translateAvailable = true;
