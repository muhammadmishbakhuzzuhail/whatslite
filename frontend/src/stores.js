import { writable, get } from "svelte/store";
import * as data from "./services/data.js";
import { t } from "./lib/i18n.js";
const tr = (k) => get(t)(k);

const params = new URLSearchParams(location.search);
let storedTheme = null;
try { storedTheme = localStorage.getItem("wa-theme"); } catch (e) {}
const initialTheme = params.get("theme") || storedTheme || "light";
const chatParam = params.get("chat");

export const chats = writable([]);
export const activeChatId = writable(null);
export const search = writable("");
export const filter = writable("Semua");
export const theme = writable(initialTheme);
theme.subscribe((v) => { try { localStorage.setItem("wa-theme", v); } catch (e) {} });

// Bahasa TUJUAN terjemahan pesan (deteksi sumber otomatis). Default: bahasa app.
let storedTrLang = null;
try { storedTrLang = localStorage.getItem("wa-tr-lang"); } catch (e) {}
export const translateLang = writable(storedTrLang || "en");
translateLang.subscribe((v) => { try { localStorage.setItem("wa-tr-lang", v); } catch (e) {} });

// Lightbox media fullscreen: {url, type:"image"|"video", caption} | null
export const lightbox = writable(null);

// Pencarian dalam satu chat (toggle dari header).
export const inChatSearch = writable(false);

// Wallpaper chat (CSS warna/gradien). Persist + terapkan ke --chat-bg.
let storedWp = null;
try { storedWp = localStorage.getItem("wa-wallpaper"); } catch (e) {}
export const wallpaper = writable(storedWp || "");
wallpaper.subscribe((v) => {
  try { localStorage.setItem("wa-wallpaper", v); } catch (e) {}
  if (typeof document !== "undefined") document.documentElement.style.setProperty("--chat-bg", v || "transparent");
});

// Suara notifikasi (WebAudio, tanpa aset). Persist.
let storedSound = null;
try { storedSound = localStorage.getItem("wa-sound"); } catch (e) {}
export const soundOn = writable(storedSound !== "0");
soundOn.subscribe((v) => { try { localStorage.setItem("wa-sound", v ? "1" : "0"); } catch (e) {} });
let _actx = null;
function playNotifSound() {
  if (!get(soundOn)) return;
  try {
    _actx = _actx || new (window.AudioContext || window.webkitAudioContext)();
    const t = _actx.currentTime;
    const o = _actx.createOscillator(), g = _actx.createGain();
    o.type = "sine";
    o.frequency.setValueAtTime(880, t);
    o.frequency.setValueAtTime(660, t + 0.09);
    g.gain.setValueAtTime(0.0001, t);
    g.gain.exponentialRampToValueAtTime(0.15, t + 0.02);
    g.gain.exponentialRampToValueAtTime(0.0001, t + 0.25);
    o.connect(g); g.connect(_actx.destination);
    o.start(t); o.stop(t + 0.26);
  } catch (e) {}
}

export const railView = writable(params.get("view") || "chats");
export const infoOpen = writable(params.get("info") === "1");
export const loggedIn = writable(data.LIVE ? false : params.get("screen") !== "qr");
export const qrImage = writable("");

// --- Kunci aplikasi (PIN) ---
function readPin() { try { return localStorage.getItem("wa-pin") || ""; } catch (e) { return ""; } }
// lockState: "off" | "locked" | "setup"
export const lockState = writable(params.get("lock") === "set" ? "setup" : readPin() ? "locked" : "off");
export const pinSet = writable(!!readPin());
export function lockNow() { if (readPin()) lockState.set("locked"); }
export function beginSetPin() { lockState.set("setup"); }
export function setPin(pin) { try { localStorage.setItem("wa-pin", pin); } catch (e) {} pinSet.set(true); lockState.set("off"); }
export function removePin() { try { localStorage.removeItem("wa-pin"); } catch (e) {} pinSet.set(false); lockState.set("off"); }
export function tryUnlock(pin) { if (pin === readPin()) { lockState.set("off"); return true; } return false; }

export function logout() {
  if (data.LIVE) data.logout();
  else { loggedIn.set(false); }
}

export const allMessages = writable({});
export const replyDraft = writable(null);
export const forwardDraft = writable(null); // {chat, idx} pesan yg akan diteruskan
export const jumpMsg = writable(null); // msgId target lompat (dari hasil pencarian)
export const newChatOpen = writable(false); // modal "chat baru / buat grup"
export const editDraft = writable(null); // {chatId, id, text} pesan yg sedang disunting

export function editMessage(chatId, msgId, text) {
  const t = (text || "").trim();
  if (!t || !msgId) return;
  data.editMessage(chatId, msgId, t);
  allMessages.update((x) => {
    const arr = [...(x[chatId] || [])];
    const i = arr.findIndex((m) => m.id === msgId);
    if (i >= 0) arr[i] = { ...arr[i], text: t, edited: true };
    return { ...x, [chatId]: arr };
  });
}
export const syncing = writable(false); // sedang tarik history dari HP

// Toast (notifikasi in-app: error kirim/koneksi). Auto-hilang.
export const toasts = writable([]);
let _toastId = 0;
export function pushToast(text, type = "error") {
  if (!text) return;
  const id = ++_toastId;
  toasts.update((a) => [...a, { id, text, type }]);
  setTimeout(() => toasts.update((a) => a.filter((t) => t.id !== id)), 4000);
}
export const chatStatus = writable({}); // jid -> subtitle (online / terakhir dilihat / N anggota)
export const typingChats = writable({}); // jid -> bool (sedang mengetik)

export function openChat(id) {
  if (id != null) data.openChat(id);
}

// (Foto profil kini lazy via /avatar/<jid> di komponen Avatar — tak perlu store.)

function nowTime() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  return `${p(d.getHours())}.${p(d.getMinutes())}`;
}

async function refreshChats() {
  chats.set(await data.getChats());
}

// Batasi memori: simpan pesan hanya utk ~6 chat terakhir dibuka (buang LRU).
const MAX_CACHED_CHATS = 6;
let _recentChats = [];
function touchChat(id) {
  _recentChats = [id, ..._recentChats.filter((x) => x !== id)].slice(0, MAX_CACHED_CHATS);
  const keep = new Set([..._recentChats, get(activeChatId)]);
  allMessages.update((m) => {
    let changed = false;
    const out = {};
    for (const k in m) {
      if (keep.has(k)) out[k] = m[k];
      else changed = true;
    }
    return changed ? out : m;
  });
}

async function reloadMessages(id) {
  if (id == null) return;
  const ms = await data.getMessages(id);
  allMessages.update((x) => ({ ...x, [id]: ms }));
  touchChat(id);
  // Sedang dibuka → tandai dibaca (read-receipt + bersihkan badge).
  if (get(activeChatId) === id) markChatRead(id);
}
export async function loadMessages(id) {
  if (id == null) return;
  if (!get(allMessages)[id]) await reloadMessages(id);
}

// Muat pesan lebih LAMA (scroll ke atas) → prepend. Kembalikan jumlah yg dimuat.
export async function loadOlder(id) {
  if (id == null || !data.LIVE) return 0;
  const cur = get(allMessages)[id] || [];
  if (!cur.length || !cur[0].ts) return 0;
  const older = await data.getMessagesBefore(id, cur[0].ts);
  if (!older.length) return 0;
  allMessages.update((x) => ({ ...x, [id]: [...older, ...(x[id] || [])] }));
  return older.length;
}

export async function sendMessage(id, text, quote, mentions) {
  const t = (text || "").trim();
  if (!t || id == null) return;
  if (data.LIVE) {
    if (mentions && mentions.length) await data.sendTextMentions(id, t, mentions);
    else if (quote && quote.id) await data.reply(id, t, quote.id, quote.senderId || "", quote.text || "");
    else await data.sendText(id, t);
    await reloadMessages(id);
    await refreshChats();
    return;
  }
  // mock: optimistik lokal
  await loadMessages(id);
  const msg = { type: "text", dir: "out", text: t, time: nowTime(), status: "sent" };
  if (quote) msg.quote = quote;
  allMessages.update((x) => ({ ...x, [id]: [...(x[id] || []), msg] }));
  chats.update((cs) => cs.map((c) => (c.id === id ? { ...c, preview: t, time: nowTime(), sent: true, unread: false, badge: 0, typing: false } : c)));
}

// Kirim media (dataURI). kind: image|video|voice|document.
export async function sendMediaMessage(id, kind, caption, fileName, dataURI, viewOnce = false) {
  if (id == null || !dataURI) return;
  if (data.LIVE) {
    await data.sendMedia(id, kind, caption || "", fileName || "", dataURI, viewOnce);
    await reloadMessages(id);
    await refreshChats();
    return;
  }
  await loadMessages(id);
  const msg = { type: kind, dir: "out", text: caption || "", thumb: dataURI, time: nowTime(), status: "sent" };
  allMessages.update((x) => ({ ...x, [id]: [...(x[id] || []), msg] }));
}

// Teruskan pesan (idx di chat sumber) ke chat tujuan.
export async function forwardMessage(srcChat, idx, toJid) {
  const m = (get(allMessages)[srcChat] || [])[idx];
  if (!m || !data.LIVE) return;
  await data.forward(srcChat, m.id, toJid);
  if (get(activeChatId) === toJid) await reloadMessages(toJid);
  await refreshChats();
}

export function deleteMessage(id, idx, everyone = false) {
  const m = (get(allMessages)[id] || [])[idx];
  if (data.LIVE && m && m.id) data.deleteMsg(id, m.id, m.senderId || "", m.dir === "out", everyone);
  allMessages.update((x) => ({ ...x, [id]: (x[id] || []).filter((_, i) => i !== idx) }));
}

// --- Mode pilih banyak pesan (bulk delete/forward) ---
export const selectMode = writable(false);
export const selectedIdx = writable([]); // array index pesan terpilih (chat aktif)
export function enterSelect(idx) {
  selectMode.set(true);
  selectedIdx.set(idx == null ? [] : [idx]);
}
export function toggleSelect(idx) {
  selectedIdx.update((a) => (a.includes(idx) ? a.filter((i) => i !== idx) : [...a, idx]));
}
export function clearSelect() {
  selectMode.set(false);
  selectedIdx.set([]);
}
export function deleteSelected(id, everyone = false) {
  const idxs = [...get(selectedIdx)].sort((a, b) => b - a); // hapus dari belakang agar index stabil
  for (const idx of idxs) deleteMessage(id, idx, everyone);
  clearSelect();
}
export function forwardSelected(id) {
  forwardDraft.set({ chat: id, idxs: [...get(selectedIdx)] });
  clearSelect();
}

export function starMessage(id, idx, on = true) {
  const m = (get(allMessages)[id] || [])[idx];
  if (data.LIVE && m && m.id) data.star(id, m.id, m.senderId || "", m.dir === "out", on);
  allMessages.update((x) => {
    const arr = [...(x[id] || [])];
    if (arr[idx]) arr[idx] = { ...arr[idx], starred: on };
    return { ...x, [id]: arr };
  });
}

// Sematkan / lepas sematan pesan di chat (optimistic + bump versi banner).
export const pinnedVersion = writable(0);
export function pinMessageAction(id, idx, on) {
  const m = (get(allMessages)[id] || [])[idx];
  if (data.LIVE && m && m.id) data.pinMessage(id, m.id, m.senderId || "", m.dir === "out", on);
  allMessages.update((x) => {
    const arr = [...(x[id] || [])];
    if (arr[idx]) arr[idx] = { ...arr[idx], pinned: on };
    return { ...x, [id]: arr };
  });
  pinnedVersion.update((n) => n + 1);
}

// Modal "Info pesan".
export const infoMsg = writable(null); // {chat, id} | null
export function showMessageInfo(id, idx) {
  const m = (get(allMessages)[id] || [])[idx];
  if (m && m.id) infoMsg.set({ chat: id, id: m.id });
}

// Reaksi = SATU reaksi milik kita per pesan (toggle), bukan increment tiap klik.
// Klik emoji sama → lepas; klik emoji beda → ganti.
export function reactMessage(id, idx, emoji) {
  const m = (get(allMessages)[id] || [])[idx];
  if (!m) return;
  const mine = (m.reactions || []).find((r) => r.mine);
  const off = mine && mine.emoji === emoji; // toggle lepas bila emoji sama
  const sent = off ? "" : emoji;            // "" = hapus reaksi di server
  if (data.LIVE && m.id) data.react(id, m.id, m.senderId || "", sent, m.dir === "out");
  allMessages.update((x) => {
    const arr = [...(x[id] || [])];
    if (!arr[idx]) return x;
    const mm = { ...arr[idx] };
    const rs = (mm.reactions || []).filter((r) => !r.mine); // buang reaksi kita yg lama
    if (sent) rs.push({ emoji: sent, count: 1, mine: true });
    mm.reactions = rs;
    arr[idx] = mm;
    return { ...x, [id]: arr };
  });
}

// Tandai chat dibaca (read-receipt) + bersihkan badge lokal.
export function markChatRead(id) {
  if (id == null) return;
  const arr = get(allMessages)[id] || [];
  const lastIn = [...arr].reverse().find((m) => m.dir === "in");
  if (data.LIVE && lastIn) data.markRead(id, lastIn.senderId || "", lastIn.id);
  chats.update((cs) => cs.map((c) => (c.id === id ? { ...c, badge: 0, unread: false } : c)));
}

// Indikator mengetik keluar (dipanggil composer saat user mengetik).
let _typingTimer = null;
export function setTyping(id, on) {
  if (id == null) return;
  data.sendTyping(id, on);
  if (on) {
    clearTimeout(_typingTimer);
    _typingTimer = setTimeout(() => data.sendTyping(id, false), 4000);
  }
}

// --- Kelola chat (pin/mute/arsip/unread/hapus) — optimistik + sinkron BE. ---
export function pinChat(id, on) {
  data.pin(id, on);
  chats.update((cs) => cs.map((c) => (c.id === id ? { ...c, pinned: on } : c)));
}
export function muteChat(id, on) {
  data.mute(id, on);
  chats.update((cs) => cs.map((c) => (c.id === id ? { ...c, muted: on } : c)));
}
export function archiveChat(id, on) {
  data.archive(id, on);
  if (on) chats.update((cs) => cs.filter((c) => c.id !== id));
  else refreshChats();
}
export function markChatUnread(id, on) {
  data.markUnread(id, on);
  chats.update((cs) => cs.map((c) => (c.id === id ? { ...c, unread: on, badge: on ? (c.badge || 1) : 0 } : c)));
}
export function removeChat(id) {
  data.deleteChat(id);
  chats.update((cs) => cs.filter((c) => c.id !== id));
  if (get(activeChatId) === id) activeChatId.set(null);
}

// --- Pencarian isi pesan (BE; "" di mock). ---
export async function searchMessages(query) {
  return data.searchMessages(query);
}

// --- Kontak & grup ---
export function blockContact(jid, on) { data.block(jid, on); }
export async function fetchContactAbout(jid) { return data.getContactAbout(jid); }
export async function fetchGroupInfo(jid) { return data.getGroupInfo(jid); }
export async function createGroup(name, participants) {
  const jid = await data.createGroup(name, participants || []);
  await refreshChats();
  return jid;
}
export function leaveGroup(jid) { data.leaveGroup(jid); }
export function setGroupSubject(jid, name) {
  data.setGroupSubject(jid, name);
  chats.update((cs) => cs.map((c) => (c.id === jid ? { ...c, name } : c)));
}
export function updateGroupParticipants(jid, members, action) { data.updateGroupParticipants(jid, members, action); }

// --- Profil sendiri ---
export function updateMyName(name) { data.setMyName(name); }
export function updateMyAbout(text) { data.setMyAbout(text); }

// --- inisialisasi (async; mock instan, LIVE via engine) ---
async function init() {
  const cs = await data.getChats();
  chats.set(cs);
  let active = null;
  if (chatParam !== "none") {
    const f = cs.find((c) => String(c.id) === String(chatParam));
    active = f ? f.id : cs[0]?.id ?? null;
  }
  activeChatId.set(active);
}

init();

if (data.LIVE) {
  data.onEvent("wa:qr", (img) => { qrImage.set(img); loggedIn.set(false); });
  data.onEvent("wa:ready", () => { loggedIn.set(true); init(); });
  data.onEvent("wa:message", async (chat) => {
    await refreshChats();
    if (get(activeChatId) === chat) reloadMessages(chat);
    // Notifikasi desktop: hanya bila chat tak aktif / window tak fokus, & tak dibisukan.
    const focused = typeof document !== "undefined" && document.hasFocus();
    if (get(activeChatId) === chat && focused) return;
    const c = get(chats).find((x) => x.id === chat);
    if (c && !c.muted) { data.notify(c.name, c.preview || "Pesan baru"); playNotifSound(); }
  });
  data.onEvent("wa:sync", () => {
    syncing.set(false);
    refreshChats();
    const a = get(activeChatId);
    if (a != null) reloadMessages(a);
  });
  data.onEvent("wa:presence", (e) => {
    if (e && e.jid) chatStatus.update((m) => ({ ...m, [e.jid]: e.text }));
  });
  data.onEvent("wa:chatinfo", (e) => {
    if (e && e.jid) chatStatus.update((m) => ({ ...m, [e.jid]: e.subtitle }));
  });
  data.onEvent("wa:typing", (e) => {
    if (!e || !e.chat) return;
    typingChats.update((m) => ({ ...m, [e.chat]: !!e.on }));
    if (e.on) setTimeout(() => typingChats.update((m) => ({ ...m, [e.chat]: false })), 6000);
  });
  data.onEvent("wa:receipt", (e) => {
    if (!e || !e.chat) return;
    const status = e.status || "read";
    const rank = { sent: 1, delivered: 2, read: 3 };
    const ids = e.ids && e.ids.length ? new Set(e.ids) : null;
    allMessages.update((all) => {
      const arr = all[e.chat];
      if (!arr) return all;
      return {
        ...all,
        [e.chat]: arr.map((m) => {
          if (m.dir !== "out") return m;
          // hanya pesan tertarget (bila ids ada); jangan turunkan status.
          if (ids && !ids.has(m.id)) return m;
          if ((rank[m.status] || 1) >= rank[status]) return m;
          return { ...m, status };
        }),
      };
    });
  });
  data.onEvent("wa:error", (e) => { console.error("WA error:", e); pushToast(typeof e === "string" ? e : tr("err_generic")); });
  data.onEvent("wa:loggedout", () => {
    loggedIn.set(false);
    syncing.set(false);
    qrImage.set("");
    chats.set([]);
    allMessages.set({});
    activeChatId.set(null);
    data.connect(); // minta QR baru untuk login/ganti akun
  });
  data.getState().then((st) => loggedIn.set(st === "ready"));
  syncing.set(true);
  setTimeout(() => syncing.set(false), 45000); // fallback bila tak ada history sync
  data.connect();
}
