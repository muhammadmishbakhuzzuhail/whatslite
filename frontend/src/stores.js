import { writable, get, derived } from "svelte/store";
import * as data from "./services/data.js";
import { t } from "./lib/i18n.js";
const tr = (k) => get(t)(k);

const params = new URLSearchParams(location.search);
let storedTheme = null;
try { storedTheme = localStorage.getItem("wa-theme"); } catch (e) {}
// Tema app: "light" | "dark" | "system" (ikuti OS).
const initialTheme = params.get("theme") || storedTheme || "system";
const chatParam = params.get("chat");

export const chats = writable([]);
export const activeChatId = writable(null);
export const search = writable("");
export const filter = writable("Semua");
export const theme = writable(initialTheme);
// Warna aksen kustom (personalisasi). "" = bawaan (#06b67f dari CSS).
let _accentInit = "";
try { _accentInit = localStorage.getItem("wa-accent") || ""; } catch (e) {}
export const accent = writable(_accentInit);
accent.subscribe((v) => { try { v ? localStorage.setItem("wa-accent", v) : localStorage.removeItem("wa-accent"); } catch (e) {} });
theme.subscribe((v) => { try { localStorage.setItem("wa-theme", v); } catch (e) {} });

// Preferensi OS (dark?) — reaktif terhadap perubahan sistem saat theme="system".
const _mql = typeof matchMedia !== "undefined" ? matchMedia("(prefers-color-scheme: dark)") : null;
export const systemDark = writable(_mql ? _mql.matches : false);
if (_mql) {
  const onCh = (e) => systemDark.set(e.matches);
  _mql.addEventListener ? _mql.addEventListener("change", onCh) : _mql.addListener(onCh);
}
// Tema efektif yg dipasang ke <html data-theme> ("light"|"dark").
export const effectiveTheme = derived([theme, systemDark], ([$t, $sd]) =>
  $t === "system" ? ($sd ? "dark" : "light") : $t
);

// Bahasa TUJUAN terjemahan pesan (deteksi sumber otomatis). Default: bahasa app.
let storedTrLang = null;
try { storedTrLang = localStorage.getItem("wa-tr-lang"); } catch (e) {}
export const translateLang = writable(storedTrLang || "en");
translateLang.subscribe((v) => { try { localStorage.setItem("wa-tr-lang", v); } catch (e) {} });

// Lightbox media fullscreen: {url, type:"image"|"video", caption} | null
export const lightbox = writable(null);

// Target pemilih reaksi (emoji penuh + search) → {chatId, idx} | null
export const reactionTarget = writable(null);

// Draft kirim media: pratinjau + caption sebelum dikirim.
// {chatId, kind, name, dataURI, viewOnce?} | null
export const mediaDraft = writable(null);

// Pencarian dalam satu chat (toggle dari header).
export const inChatSearch = writable(false);

// Suara notifikasi (WebAudio, tanpa aset). Persist.
let storedSound = null;
try { storedSound = localStorage.getItem("wa-sound"); } catch (e) {}
export const soundOn = writable(storedSound !== "0");
soundOn.subscribe((v) => { try { localStorage.setItem("wa-sound", v ? "1" : "0"); } catch (e) {} });

// Anti-delete: tampilkan isi pesan yg ditarik pengirim + tag "dihapus". Default ON.
let storedShowDel = null;
try { storedShowDel = localStorage.getItem("wa-show-deleted"); } catch (e) {}
export const showDeleted = writable(storedShowDel !== "0");
showDeleted.subscribe((v) => { try { localStorage.setItem("wa-show-deleted", v ? "1" : "0"); } catch (e) {} });
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
// Pindah ke section non-chat (Status/Saluran/Komunitas) → tutup chat aktif agar
// panel kanan tak menampilkan grup yang tak nyambung dgn sidebar.
railView.subscribe((v) => {
  if (v === "status" || v === "channels" || v === "communities" || v === "calls" || v === "storage" || v === "scheduled") activeChatId.set(null);
});

// --- Panggilan (signaling-only: log + tolak; tak ada media call) ---
export const calls = writable([]);          // log panggilan (CallsPane)
export const incomingCall = writable(null); // {id,jid,name,video,group} → banner masuk
export async function refreshCalls() { calls.set(await data.getCalls()); }
export function rejectIncomingCall(c) {
  if (!c) return;
  data.rejectCall(c.jid, c.id);
  incomingCall.set(null);
  refreshCalls();
}
export function dismissIncomingCall() { incomingCall.set(null); }

// --- Dialog input teks (prompt() native TAK jalan di WebKitGTK) ---
export const promptDialog = writable(null); // {title, value, onYes}
export function askPrompt(title, def, onYes) { promptDialog.set({ title, value: def || "", onYes }); }
export function resolvePrompt(ok, val) {
  const d = get(promptDialog);
  promptDialog.set(null);
  if (ok && d && d.onYes && (val || "").trim()) d.onYes(val.trim());
}

// --- Terjemahan: cache per-pesan (bertahan saat scroll/reload) + auto per-chat ---
export const translations = writable({}); // msgId -> teks terjemahan
export function setTranslation(id, text) { translations.update((m) => ({ ...m, [id]: text })); }
export function clearTranslation(id) { translations.update((m) => { const n = { ...m }; delete n[id]; return n; }); }
let _autoTrInit = [];
try { _autoTrInit = JSON.parse(localStorage.getItem("wa-autotranslate") || "[]") || []; } catch (e) {}
export const autoTranslateChats = writable(new Set(_autoTrInit));
export function toggleAutoTranslate(jid) {
  autoTranslateChats.update((s) => {
    const n = new Set(s);
    n.has(jid) ? n.delete(jid) : n.add(jid);
    try { localStorage.setItem("wa-autotranslate", JSON.stringify([...n])); } catch (e) {}
    return n;
  });
}

// --- Folder/filter chat kustom (lokal) ---
let _foldInit = [];
try { _foldInit = JSON.parse(localStorage.getItem("wa-folders") || "[]") || []; } catch (e) {}
export const folders = writable(_foldInit);
function persistFolders(v) { try { localStorage.setItem("wa-folders", JSON.stringify(v)); } catch (e) {} }
export function addFolder(name) { folders.update((f) => { if (f.some((x) => x.name === name)) return f; const n = [...f, { name, jids: [] }]; persistFolders(n); return n; }); }
export function deleteFolder(name) { folders.update((f) => { const n = f.filter((x) => x.name !== name); persistFolders(n); return n; }); }
export function toggleChatFolder(name, jid) {
  folders.update((f) => {
    const n = f.map((x) => x.name === name ? { ...x, jids: x.jids.includes(jid) ? x.jids.filter((j) => j !== jid) : [...x.jids, jid] } : x);
    persistFolders(n); return n;
  });
}
export const folderPickFor = writable(null); // jid chat yg sedang diatur foldernya

// --- Dialog konfirmasi (confirm()/prompt() native TAK jalan di WebKitGTK) ---
export const confirmDialog = writable(null); // {text, onYes}
export function askConfirm(text, onYes) { confirmDialog.set({ text, onYes }); }
export function resolveConfirm(ok) {
  const d = get(confirmDialog);
  confirmDialog.set(null);
  if (ok && d && d.onYes) d.onYes();
}
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

// --- Panel profil kontak (klik mention / pengirim grup) ---
export const profileJid = writable(null); // jid kontak yg profilnya terbuka
export function openProfile(jid) { if (jid) profileJid.set(jid); }
export function closeProfile() { profileJid.set(null); }
// "Pesan" dari panel profil → buka/buat chat dgn kontak itu.
export function messageContact(jid) {
  if (!jid) return;
  profileJid.set(null);
  activeChatId.set(jid);
  openChat(jid);
}
// Simpan/hapus label nama lokal (app, BUKAN sync ke HP/WA) → wa:sync refresh nama.
export function saveContactLabel(jid, name) { data.saveContactLabel(jid, name); }
export function removeContactLabel(jid) { data.removeContactLabel(jid); }

// (Foto profil kini lazy via /avatar/<jid> di komponen Avatar — tak perlu store.)

// --- Draf teks per-chat ---
// Teks yang sudah diketik tapi belum dikirim disimpan saat pindah chat
// (localStorage), dipulihkan saat chat dibuka lagi, & tampil "Draf: …" di sidebar.
let _draftInit = {};
try { _draftInit = JSON.parse(localStorage.getItem("wa-drafts") || "{}") || {}; } catch (e) {}
export const drafts = writable(_draftInit);
export function setDraft(chatId, text) {
  if (!chatId) return;
  drafts.update((d) => {
    const n = { ...d };
    if (text && text.trim()) n[chatId] = text;
    else delete n[chatId];
    try { localStorage.setItem("wa-drafts", JSON.stringify(n)); } catch (e) {}
    return n;
  });
}
export function getDraft(chatId) { return get(drafts)[chatId] || ""; }

// --- Wallpaper per-chat (lokal) ---
let _wpInit = {};
try { _wpInit = JSON.parse(localStorage.getItem("wa-wallpaper") || "{}") || {}; } catch (e) {}
export const wallpapers = writable(_wpInit);
export function setWallpaper(jid, val) {
  if (!jid) return;
  wallpapers.update((w) => {
    const n = { ...w };
    if (val) n[jid] = val; else delete n[jid];
    try { localStorage.setItem("wa-wallpaper", JSON.stringify(n)); } catch (e) {}
    return n;
  });
}

// Kosongkan isi chat (hapus semua pesan, chat tetap ada).
export function clearChatMessages(jid) {
  if (!jid) return;
  data.clearChat(jid);
  allMessages.update((m) => { const n = { ...m }; delete n[jid]; return n; });
}

function nowTime() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  return `${p(d.getHours())}.${p(d.getMinutes())}`;
}

async function refreshChats() {
  chats.set(await data.getChats());
}
// Debounce refresh sidebar (receipt grup = puluhan event → 1 query saja).
let _chatRefreshTimer = null;
function scheduleChatRefresh() {
  clearTimeout(_chatRefreshTimer);
  _chatRefreshTimer = setTimeout(refreshChats, 500);
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

// Gabung jendela "terbaru" (fresh, 200 terakhir, otoritatif utk status/edit)
// dgn array saat ini — JANGAN buang pesan lama hasil loadOlder (yg sedang
// dibaca saat scroll-up). Tanpa ini, pesan masuk meng-overwrite → array
// menciut + posisi loncat ke bawah (auto-scroll padahal user scroll-up).
function mergeMessages(cur, fresh) {
  if (!cur || !cur.length) return fresh;
  const byId = new Map();
  for (const m of cur) if (m && m.id) byId.set(m.id, m);
  for (const m of fresh) if (m && m.id) byId.set(m.id, m); // fresh menang (status/edit terbaru)
  return [...byId.values()].sort((a, b) => (a.ts || 0) - (b.ts || 0));
}
async function reloadMessages(id) {
  if (id == null) return;
  const ms = await data.getMessages(id);
  allMessages.update((x) => ({ ...x, [id]: mergeMessages(x[id], ms) }));
  touchChat(id);
  // Sedang dibuka → tandai dibaca (read-receipt + bersihkan badge).
  if (get(activeChatId) === id) markChatRead(id);
}
export async function loadMessages(id) {
  if (id == null) return;
  if (!get(allMessages)[id]) await reloadMessages(id);
  else touchChat(id); // cache-hit: tetap segarkan LRU agar tak ke-evict keliru
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
export async function sendMediaMessage(id, kind, caption, fileName, dataURI, viewOnce = false, seconds = 0) {
  if (id == null || !dataURI) return;
  if (data.LIVE) {
    await data.sendMedia(id, kind, caption || "", fileName || "", dataURI, viewOnce, seconds);
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
  data.onEvent("wa:ready", () => { loggedIn.set(true); init(); refreshCalls(); });
  data.onEvent("wa:call", (c) => { if (c) { incomingCall.set(c); refreshCalls(); playNotifSound(); } });
  data.onEvent("wa:reminder", (r) => { if (r) { pushToast("🔔 " + (r.chatName || "") + (r.note ? ": " + r.note : ""), "ok"); playNotifSound(); } });
  data.onEvent("wa:callupdate", () => refreshCalls());
  data.onEvent("wa:message", async (chat) => {
    await refreshChats();
    if (get(activeChatId) === chat) reloadMessages(chat);
    // Notifikasi desktop: hanya bila chat tak aktif / window tak fokus, & tak dibisukan.
    const focused = typeof document !== "undefined" && document.hasFocus();
    if (get(activeChatId) === chat && focused) return;
    const c = get(chats).find((x) => x.id === chat);
    if (c && !c.muted) { data.notify(c.name, c.preview || tr("new_message")); playNotifSound(); }
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
    // value: nama (grup) | true (1:1) saat mengetik; false saat berhenti.
    const v = e.on ? (e.who || true) : false;
    typingChats.update((m) => ({ ...m, [e.chat]: v }));
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
    scheduleChatRefresh(); // perbarui centang di sidebar (debounce → receipt grup banyak)
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
