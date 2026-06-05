// Data palsu untuk pratinjau UI. Nanti diganti data nyata dari engine Go
// (via services/engine.js + Wails). Bentuk objek sengaja dekat ke model nyata.

export const me = {
  name: "Zuhail",
  about: "Hai! Saya pakai WhatsApp.",
  phone: "+62 812-3456-7890",
  color: "#6a9e3d",
};

export const archivedCount = 3;

export const chats = [
  { id: 1, name: "Andi Pratama", color: "#6a9e3d", time: "19.08", preview: "Mantap! Sampai nanti malam 🙌", sent: true, status: "online", pinned: true, phone: "+62 813-1111-2222", about: "Sibuk" },
  { id: 6, name: "Mama", color: "#c95a8b", time: "12.30", preview: "Sudah sampai rumah?", sent: true, pinned: true, status: "online", phone: "+62 813-9999-0000", about: "❤️ Keluarga" },
  { id: 2, name: "Keluarga 👨‍👩‍👧", color: "#e0794f", time: "18.41", preview: "Ibu: Jangan lupa makan ya nak", badge: 2, unread: true, group: true, status: "5 anggota",
    members: "Kamu, Ayah, Ibu, Kakak, Adik", about: "Grup keluarga besar",
    pinnedMsg: { sender: "Ibu", text: "📍 Lokasi kumpul lebaran • Min, 8 Jun" } },
  { id: 3, name: "Sarah", color: "#b86ac9", time: "17.55", preview: "Oke besok aku kabarin lagi", sent: true, status: "terakhir dilihat hari ini 18.00", phone: "+62 856-2222-3333", about: "Available" },
  { id: 4, name: "Tim Proyek X", color: "#3d8bd3", time: "16.20", preview: "Budi: file-nya udah aku upload", badge: 12, unread: true, group: true, typing: true, status: "8 anggota", members: "Kamu, Budi, Sinta, Reza, +4", about: "Koordinasi proyek X" },
  { id: 5, name: "Rian", color: "#2aa89e", time: "14.03", preview: "Haha iya bener banget 😂", muted: true, status: "terakhir dilihat hari ini 13.50", phone: "+62 877-4444-5555", about: "Ngoding terus" },
  { id: 7, name: "Info Kampus", color: "#5a6ac9", time: "Kemarin", preview: "Pengumuman: jadwal UAS telah...", muted: true, group: true, status: "124 anggota", members: "124 anggota", about: "Info resmi kampus" },
  { id: 8, name: "Dimas", color: "#d8902a", time: "Kemarin", preview: "Nanti aku telpon ya", sent: true, status: "terakhir dilihat kemarin", phone: "+62 821-6666-7777", about: "Di luar kota" },
  { id: 9, name: "Grup Futsal", color: "#6a9e3d", time: "Kemarin", preview: "Anto: yang ikut absen dong", badge: 5, unread: true, group: true, status: "15 anggota", members: "15 anggota", about: "Futsal tiap Jumat" },
  { id: 10, name: "Nadia", color: "#e0794f", time: "Senin", preview: "Makasih banyak ya 🙏", sent: true, status: "terakhir dilihat Senin", phone: "+62 838-8888-9999", about: "Terima kasih 🙏" },
  { id: 11, name: "Kerja Kelompok", color: "#3d8bd3", time: "Senin", preview: "Kamu: oke aku kerjain bagian 2", sent: true, group: true, status: "4 anggota", members: "Kamu, Dewi, Tari, Joko", about: "Tugas akhir semester" },
  { id: 12, name: "Bayu", color: "#b86ac9", time: "Minggu", preview: "📷 Foto", muted: true, status: "terakhir dilihat Minggu", phone: "+62 852-1010-2020", about: "Fotografi 📷" },
];

// warna nama pengirim grup (konsisten per nama)
const C = { Ayah: "#e0794f", Ibu: "#b86ac9", Kakak: "#3d8bd3", Adik: "#2aa89e", Budi: "#d8902a" };

export const messagesByChat = {
  1: [
    { type: "day" },
    { type: "system" },
    { type: "text", dir: "in", text: "Halo! Jadi nanti malam ngumpul jam berapa?", time: "19.02", reactions: [{ emoji: "👍", count: 1 }] },
    { type: "text", dir: "out", text: "Jam 8 ya, di tempat biasa 👌", time: "19.03", status: "read" },
    { type: "text", dir: "in", text: "Oke sip. Oh iya aku bawa kamera, sekalian foto-foto. Kamu bawa speaker yang kemarin gak?", time: "19.04" },
    { type: "text", dir: "out", text: "Bawa dong, udah aku charge full 🔋", time: "19.05", status: "read", quote: { name: "Andi Pratama", text: "Kamu bawa speaker yang kemarin gak?" } },
    { type: "image", dir: "in", caption: "Spot kemarin, bagus banget buat sunset 🌅", time: "19.06", forwarded: true, thumb: "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='320' height='240'%3E%3Cdefs%3E%3ClinearGradient id='s' x1='0' y1='0' x2='0' y2='1'%3E%3Cstop offset='0' stop-color='rgb(255,176,99)'/%3E%3Cstop offset='0.45' stop-color='rgb(255,118,92)'/%3E%3Cstop offset='1' stop-color='rgb(108,58,128)'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='320' height='240' fill='url(%23s)'/%3E%3Ccircle cx='160' cy='128' r='38' fill='rgb(255,228,158)'/%3E%3Crect x='120' y='148' width='80' height='4' rx='2' fill='rgba(255,228,158,0.6)'/%3E%3Crect x='132' y='158' width='56' height='3' rx='1.5' fill='rgba(255,228,158,0.4)'/%3E%3Cpath d='M0 196 Q80 168 160 190 T320 184 V240 H0 Z' fill='rgba(70,36,82,0.55)'/%3E%3Cpath d='M0 214 Q90 196 180 214 T320 210 V240 H0 Z' fill='rgba(34,18,46,0.72)'/%3E%3C/svg%3E" },
    { type: "voice", dir: "in", duration: "0:12", time: "19.07" },
    { type: "text", dir: "out", text: "Mantap! Sampai nanti malam 🙌", time: "19.08", status: "sending" },
  ],
  2: [
    { type: "day" },
    { type: "system" },
    { type: "text", dir: "in", sender: "Ayah", senderColor: C.Ayah, text: "Nanti malam kita makan di luar ya 🍽️", time: "18.30" },
    { type: "text", dir: "in", sender: "Ibu", senderColor: C.Ibu, text: "Setuju! Mau makan apa?", time: "18.35" },
    { type: "text", dir: "in", sender: "Ibu", senderColor: C.Ibu, text: "Sushi atau Padang?", time: "18.35" },
    { type: "text", dir: "out", text: "Sushi aja yuk 🍣", time: "18.36", status: "read" },
    { type: "image", dir: "in", sender: "Kakak", senderColor: C.Kakak, caption: "Tempatnya di sini ya 📍", time: "18.40" },
    { type: "unread", count: 2 },
    { type: "text", dir: "in", sender: "Ibu", senderColor: C.Ibu, text: "Jangan lupa makan ya nak", time: "18.41", reactions: [{ emoji: "❤️", count: 2 }] },
  ],
  3: [
    { type: "day" },
    { type: "system" },
    { type: "text", dir: "out", text: "Sar, jadi ketemu besok?", time: "17.50", status: "delivered" },
    { type: "text", dir: "in", text: "Hmm belum pasti nih, masih nunggu kabar", time: "17.53" },
    { type: "text", dir: "out", text: "Oke santai aja", time: "17.54", status: "read" },
    { type: "text", dir: "in", text: "Oke besok aku kabarin lagi", time: "17.55" },
  ],
};

export const defaultMessages = [
  { type: "system" },
  { type: "text", dir: "in", text: "Hai 👋", time: "12.00" },
  { type: "text", dir: "out", text: "Halo, apa kabar?", time: "12.01", status: "read" },
];

// name/desc diterjemahkan via i18n (key → key + "_d").
export const settingsItems = [
  { icon: "key", key: "account" },
  { icon: "lock", key: "privacy" },
  { icon: "chat", key: "chats_set" },
  { icon: "bell", key: "notifications" },
  { icon: "disk", key: "storage" },
  { icon: "help", key: "help" },
];
