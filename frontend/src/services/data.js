// ============================================================
// services/data.js — SATU jembatan UI ↔ sumber data.
// Mock (browser) ATAU engine Go nyata (app native Wails) — deteksi otomatis.
// ============================================================

import {
  chats as mockChats,
  messagesByChat,
  defaultMessages,
  me as mockMe,
  settingsItems as mockSettings,
  archivedCount as mockArchived,
} from "../lib/data/mock.js";

const A = typeof window !== "undefined" ? window.go?.app?.App : null;
export const LIVE = !!(A && A.GetChats); // true di app native, false di browser

// Mode ?data=real (browser): baca snapshot data nyata (real-data.json hasil
// `whatsapp-lite --export-json`) menggantikan mock → screenshot data nyata + UI
// nyata tanpa display server. Tak aktif di app native (pakai engine langsung).
const _params = typeof location !== "undefined" ? new URLSearchParams(location.search) : new URLSearchParams();
export const REAL = !LIVE && _params.get("data") === "real";
let _realCache = null;
async function loadReal() {
  if (_realCache) return _realCache;
  try {
    _realCache = await fetch("./real-data.json").then((r) => r.json());
  } catch (e) {
    _realCache = { chats: [], messages: {}, me: mockMe };
  }
  return _realCache;
}

const PALETTE = ["#6a9e3d", "#e0794f", "#b86ac9", "#3d8bd3", "#2aa89e", "#c95a8b", "#5a6ac9", "#d8902a"];
function hash(s) {
  let h = 0;
  for (const ch of s || "") h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return h;
}
export function colorFor(s) {
  return PALETTE[hash(s) % PALETTE.length];
}

// Palet warna nama pengirim grup — cerah & distinct (ala WhatsApp), terbaca di
// bubble terang maupun gelap. Hash pakai JID pengirim (stabil) bukan nama.
const SENDER_PALETTE = [
  "#e5446d", "#00a884", "#6a5acd", "#ff8c42", "#2196f3", "#c2185b", "#009688",
  "#7cb342", "#8e24aa", "#f4511e", "#00acc1", "#5e35b1", "#d81b60", "#43a047",
];
export function senderColorFor(key) {
  return SENDER_PALETTE[hash(key) % SENDER_PALETTE.length];
}

// URL foto profil via asset-server (cache FILE, lazy — hanya avatar yg dirender).
// "" di mock/preview → komponen fallback ke inisial. Native → /avatar/<jid>.
export function avatarUrl(jid) {
  return LIVE && jid ? "/avatar/" + encodeURIComponent(jid) : "";
}

// Lengkapi pesan: warna pengirim + objek quote (balasan).
function mapMsg(m) {
  const o = { ...m, senderColor: senderColorFor(m.senderId || m.sender) };
  if (m.quoteText || m.quoteName) o.quote = { name: m.quoteName || "", text: m.quoteText || "", id: m.quoteId || "" };
  return o;
}

export async function getChats() {
  if (REAL) {
    const d = await loadReal();
    return (d.chats || []).map((c) => ({ ...c, color: colorFor(c.id) }));
  }
  if (!LIVE) return mockChats;
  const cs = (await A.GetChats()) || [];
  return cs.map((c) => ({ ...c, color: colorFor(c.id) }));
}

export async function getMessages(id) {
  if (REAL) {
    const d = await loadReal();
    const ms = (d.messages && d.messages[id]) || [];
    return ms.map(mapMsg);
  }
  if (!LIVE) return messagesByChat[id] || defaultMessages;
  const ms = (await A.GetMessages(id)) || [];
  return ms.map(mapMsg);
}

// Pesan lebih lama dari beforeTs (pagination scroll atas). [] di mock/preview.
export async function getMessagesBefore(id, beforeTs) {
  if (!LIVE) return [];
  const ms = (await A.GetMessagesBefore(id, beforeTs)) || [];
  return ms.map(mapMsg);
}

export function getProfile() {
  if (REAL && _realCache && _realCache.me) return _realCache.me;
  return mockMe;
}
// Profil akun (async) — native ambil dari engine.
export async function fetchProfile() {
  if (LIVE) {
    const p = (await A.GetProfile()) || {};
    return { name: p.name || "Saya", phone: p.phone || "", about: p.about || "", color: colorFor(p.phone || p.name || "me") };
  }
  return getProfile();
}
export function getSettingsItems() {
  return mockSettings;
}
export function getArchivedCount() {
  if (REAL) return 0;
  if (!LIVE) return mockArchived;
  return 0;
}
export async function getArchivedChats() {
  if (LIVE) {
    const cs = (await A.GetArchivedChats()) || [];
    return cs.map((c) => ({ ...c, color: colorFor(c.id) }));
  }
  return [];
}

// Kirim pesan: di LIVE lewat engine Go; di mock biarkan store lokal yang urus.
export async function sendText(id, text) {
  if (LIVE) return A.SendText(id, text);
}

export function connect() {
  if (LIVE) A.Connect();
}
export function logout() {
  if (LIVE) A.Logout();
}
export function openChat(jid) {
  if (LIVE) A.OpenChat(jid);
}
export async function getState() {
  if (LIVE) return A.GetState();
  return "ready";
}

// --- Aksi pesan (Tier 1) ---
export async function sendMedia(jid, kind, caption, fileName, dataURI, viewOnce = false, seconds = 0) {
  if (LIVE) return A.SendMedia(jid, kind, caption, fileName, dataURI, viewOnce, seconds);
  return "";
}
export async function sendContact(jid, displayName, phone) {
  if (LIVE) return A.SendContact(jid, displayName, phone);
  return "";
}
export async function sendGif(jid, dataURI) {
  if (LIVE) return A.SendGif(jid, dataURI);
  return "";
}
export async function sendSticker(jid, dataURI) {
  if (LIVE) return A.SendSticker(jid, dataURI);
  return "";
}
export function editMessage(jid, msgID, text) {
  if (LIVE) A.EditMessage(jid, msgID, text);
}
export async function sendTextMentions(jid, text, mentions) {
  if (LIVE) return A.SendTextMentioned(jid, text, mentions);
  return "";
}
export async function reply(jid, text, quotedID, quotedSender, quotedText) {
  if (LIVE) return A.Reply(jid, text, quotedID, quotedSender, quotedText);
  return "";
}
export async function forward(srcChat, msgID, toJID) {
  if (LIVE) return A.Forward(srcChat, msgID, toJID);
  return "";
}
export function react(chat, msgID, sender, emoji, fromMe) {
  if (LIVE) A.React(chat, msgID, sender, emoji, fromMe);
}
export function deleteMsg(chat, msgID, sender, fromMe, everyone) {
  if (LIVE) A.DeleteMessage(chat, msgID, sender, fromMe, everyone);
}
export function star(chat, msgID, sender, fromMe, on) {
  if (LIVE) A.StarMessage(chat, msgID, sender, fromMe, on);
}
export function markRead(chat, sender, msgID) {
  if (LIVE) A.MarkRead(chat, sender, msgID);
}
export function pinMessage(chat, msgID, sender, fromMe, on) {
  if (LIVE) A.PinMessage(chat, msgID, sender, fromMe, on);
}
export async function getPinned(chat) {
  if (LIVE) return (await A.GetPinned(chat)) || [];
  return [];
}
export async function getMessageInfo(chat, msgID) {
  if (LIVE) return A.GetMessageInfo(chat, msgID);
  return null;
}
export function sendTyping(jid, composing) {
  if (LIVE) A.SendTyping(jid, composing);
}
export async function sendLocation(jid, lat, lng, name) {
  if (LIVE) return A.SendLocation(jid, lat, lng, name);
  return "";
}
export async function sendPoll(jid, question, options, selectable) {
  if (LIVE) return A.SendPoll(jid, question, options, selectable);
  return "";
}
export function setDisappearing(jid, seconds) {
  if (LIVE) A.SetDisappearing(jid, seconds);
}
export function votePoll(chat, pollSender, pollID, options) {
  if (LIVE) A.VotePoll(chat, pollSender, pollID, options);
}
export async function getPollVotes(pollID) {
  if (LIVE) return (await A.GetPollVotes(pollID)) || { counts: {}, total: 0 };
  return { counts: {}, total: 0 };
}
export async function exportChat(jid) {
  if (LIVE) return A.ExportChat(jid);
  return "";
}

// --- Kelola chat (Tier 2) ---
export function pin(jid, on) {
  if (LIVE) A.Pin(jid, on);
}
export function mute(jid, on) {
  if (LIVE) A.Mute(jid, on);
}
export function archive(jid, on) {
  if (LIVE) A.Archive(jid, on);
}
export function markUnread(jid, on) {
  if (LIVE) A.MarkUnread(jid, on);
}
export function deleteChat(jid) {
  if (LIVE) A.DeleteChat(jid);
}
export function clearChat(jid) {
  if (LIVE) A.ClearChat(jid);
}
export async function searchMessages(query) {
  if (LIVE) return (await A.SearchMessages(query)) || [];
  return [];
}
export async function getStarred() {
  if (LIVE) return (await A.GetStarred()) || [];
  return [];
}

// --- Panggilan (log + tolak; tak ada media call) ---
export async function getCalls() {
  if (LIVE) return (await A.GetCalls()) || [];
  return [];
}
export function rejectCall(jid, callID) {
  if (LIVE) A.RejectCall(jid, callID);
}

// --- Setelan: retensi pesan (hari; 0 = selamanya) ---
export async function getRetention() {
  if (LIVE) return await A.GetRetention();
  return 90;
}
export function setRetention(days) {
  if (LIVE) A.SetRetention(days);
}

// --- Status / Stories ---
export async function getStatuses() {
  if (LIVE) return (await A.GetStatuses()) || [];
  return [];
}
export function reactStatus(posterJid, statusID, emoji) { if (LIVE) A.ReactStatus(posterJid, statusID, emoji); }
export function replyStatus(posterJid, statusID, statusText, text) { if (LIVE) A.ReplyStatus(posterJid, statusID, statusText, text); }
export async function postTextStatus(text, bgArgb = 0, font = 0) {
  if (LIVE) return await A.PostTextStatus(text, bgArgb, font);
  return "";
}
export async function postMediaStatus(kind, caption, dataURI) {
  if (LIVE) return A.PostMediaStatus(kind, caption, dataURI);
  return "";
}
export async function getStatusViewers(statusID) {
  if (LIVE) return (await A.GetStatusViewers(statusID)) || [];
  return [];
}

// --- Channels / Saluran (newsletter, read-only) ---
export async function getChannels() {
  if (LIVE) return (await A.GetChannels()) || [];
  return [];
}
export async function getChannelMessages(jid) {
  if (LIVE) return (await A.GetChannelMessages(jid)) || [];
  return [];
}
export async function followChannel(link) {
  if (LIVE) return A.FollowChannel(link);
  return null;
}
export async function getRecommendedChannels(query) {
  if (LIVE) return (await A.GetRecommendedChannels(query || "")) || [];
  return [];
}
export function followChannelByJID(jid) {
  if (LIVE) A.FollowChannelByJID(jid);
}
export function unfollowChannel(jid) {
  if (LIVE) A.UnfollowChannel(jid);
}
export function muteChannel(jid, on) {
  if (LIVE) A.MuteChannel(jid, on);
}
export function reactChannel(channelJID, msgID, serverID, emoji) {
  if (LIVE) A.ReactChannel(channelJID, msgID, serverID, emoji);
}

// --- Communities / Komunitas ---
export async function getCommunities() {
  if (LIVE) return (await A.GetCommunities()) || [];
  return [];
}
export function leaveCommunity(jid) {
  if (LIVE) A.LeaveCommunity(jid);
}

// --- Kontak & profil (Tier 4) ---
export async function getContactAbout(jid) {
  if (LIVE) return A.GetContactAbout(jid);
  return "";
}
// Profil kontak utk panel (klik mention / pengirim grup).
export async function getContactProfile(jid) {
  if (LIVE) return A.GetContactProfile(jid);
  const c = (await getChats()).find((x) => x.id === jid);
  return { jid, name: c?.name || jid, phone: c?.phone || "", about: "", saved: !!c };
}
// Simpan nama lokal (label app, BUKAN sync ke HP/WA).
export function saveContactLabel(jid, name) {
  if (LIVE) A.SaveContactLabel(jid, name);
}
export function subscribePresence(jid) {
  if (LIVE) A.SubscribePresence(jid);
}
export function removeContactLabel(jid) {
  if (LIVE) A.RemoveContactLabel(jid);
}
export function block(jid, on) {
  if (LIVE) A.Block(jid, on);
}
export async function getBlockedContacts() {
  if (LIVE) return (await A.GetBlockedContacts()) || [];
  return [];
}
// Pencarian GIF lewat backend (hindari CORS WebKitGTK). "" = trending.
// pos = kursor halaman (infinite scroll). Kembalikan {items, next}.
export async function searchGifs(query, pos = "") {
  if (LIVE) return (await A.SearchGifs(query || "", pos || "")) || { items: [], next: "" };
  try {
    const KEY = "LIVDSRZULELA";
    const base = query ? `https://g.tenor.com/v1/search?q=${encodeURIComponent(query)}` : `https://g.tenor.com/v1/trending`;
    const r = await fetch(`${base}&key=${KEY}&limit=50&media_filter=minimal&contentfilter=high${pos ? `&pos=${encodeURIComponent(pos)}` : ""}`).then((x) => x.json());
    const items = (r.results || []).map((g) => { const m = (g.media && g.media[0]) || {}; return { id: g.id, preview: m.tinygif?.url, mp4: m.mp4?.url }; }).filter((g) => g.preview && g.mp4);
    return { items, next: r.next || "" };
  } catch (e) { return { items: [], next: "" }; }
}
// Pencarian stiker transparan (Tenor) lewat backend — tab "Online" picker stiker.
export async function searchStickers(query, pos = "") {
  if (LIVE) return (await A.SearchStickers(query || "", pos || "")) || { items: [], next: "" };
  return { items: [], next: "" };
}
// Daftar kontak (buku-alamat + label lokal) utk panel Kontak sidebar.
export async function getContacts() {
  if (LIVE) return (await A.GetContacts()) || [];
  return (await getChats()).filter((c) => !c.group).map((c) => ({ jid: c.id, name: c.name, phone: c.phone || "", saved: true }));
}
export async function getPrivacy() {
  if (LIVE) return (await A.GetPrivacy()) || {};
  return {};
}
export function setPrivacy(name, value) {
  if (LIVE) A.SetPrivacy(name, value);
}
export function setMyName(name) {
  if (LIVE) A.SetMyName(name);
}
export function setMyAbout(text) {
  if (LIVE) A.SetMyAbout(text);
}
// Ganti foto profil sendiri (data-URI JPEG; di-encode di FE via canvas).
export function setMyPhoto(fullURI, previewURI) {
  if (LIVE) A.SetMyPhoto(fullURI, previewURI || "");
}
export async function createChannel(name, desc) { if (LIVE) return await A.CreateChannel(name, desc || ""); return ""; }
export function postChannel(jid, text) { if (LIVE) A.PostChannel(jid, text); }
export async function getBusinessProfile(jid) { if (LIVE) return await A.GetBusinessProfile(jid); return { isBiz: false }; }
export async function getCommonGroups(jid) { if (LIVE) return (await A.GetCommonGroups(jid)) || []; return []; }
export async function getStorageUsage() { if (LIVE) return await A.GetStorageUsage(); return { dbBytes: 0, mediaBytes: 0, msgCount: 0, kinds: [] }; }
// --- Pesan terjadwal + pengingat (client-side) ---
export function scheduleMessage(jid, text, sendAt) { if (LIVE) A.ScheduleMessage(jid, text, sendAt); }
export async function getScheduled() { if (LIVE) return (await A.GetScheduled()) || []; return []; }
export function cancelScheduled(id) { if (LIVE) A.CancelScheduled(id); }
export function addReminder(jid, msgId, note, remindAt) { if (LIVE) A.AddReminder(jid, msgId, note, remindAt); }
export async function getReminders() { if (LIVE) return (await A.GetReminders()) || []; return []; }
export function cancelReminder(id) { if (LIVE) A.CancelReminder(id); }
export async function getProxy() { if (LIVE) return await A.GetProxy(); return ""; }
export function setProxy(addr) { if (LIVE) A.SetProxy(addr); }
export async function getBackgroundClose() { if (LIVE) return await A.GetBackgroundClose(); return false; }
export function setBackgroundClose(on) { if (LIVE) A.SetBackgroundClose(on); }
export function quitApp() { if (LIVE) A.Quit(); }
export function setUnreadBadge(n) { if (LIVE) A.SetUnreadBadge(n); }
export async function addViaQR(code) { if (LIVE) return await A.AddViaQR(code); return ""; }

// --- Grup (Tier 5) ---
export async function getGroupInfo(jid) {
  if (LIVE) return A.GetGroupInfo(jid);
  return null;
}
export async function createGroup(name, participants) {
  if (LIVE) return A.CreateGroup(name, participants);
  return "";
}
export function leaveGroup(jid) {
  if (LIVE) A.LeaveGroup(jid);
}
export function setGroupSubject(jid, name) {
  if (LIVE) A.SetGroupSubject(jid, name);
}
export function setGroupDescription(jid, topic) {
  if (LIVE) A.SetGroupDescription(jid, topic);
}
export function updateGroupParticipants(jid, members, action) {
  if (LIVE) A.UpdateGroupParticipants(jid, members, action);
}
export async function groupInviteLink(jid, reset = false) {
  if (LIVE) return A.GroupInviteLink(jid, reset);
  return "";
}
export function setGroupPhoto(jid, dataURI) {
  if (LIVE) A.SetGroupPhoto(jid, dataURI);
}
// --- Setelan admin grup ---
export function setGroupAnnounce(jid, on) { if (LIVE) A.SetGroupAnnounce(jid, on); }
export function setGroupLocked(jid, on) { if (LIVE) A.SetGroupLocked(jid, on); }
export function setGroupJoinApproval(jid, on) { if (LIVE) A.SetGroupJoinApproval(jid, on); }
export function setGroupAddMode(jid, adminOnly) { if (LIVE) A.SetGroupAddMode(jid, adminOnly); }
export async function getGroupRequests(jid) { if (LIVE) return (await A.GetGroupRequests(jid)) || []; return []; }
export function updateGroupRequest(jid, members, approve) { if (LIVE) A.UpdateGroupRequest(jid, members, approve); }
export async function getChatMedia(jid) { if (LIVE) return (await A.GetChatMedia(jid)) || []; return []; }
export async function joinGroupLink(link) { if (LIVE) return await A.JoinGroupLink(link); return ""; }
export async function previewGroupLink(link) { if (LIVE) return await A.PreviewGroupLink(link); return ""; }
// Cek nomor terdaftar di WhatsApp (sebelum mulai chat / simpan kontak).
export async function isOnWhatsApp(phones) { if (LIVE) return (await A.IsOnWhatsApp(phones)) || []; return []; }
// Login via nomor telepon (kode 8-char alternatif QR).
export async function linkWithPhone(phone) { if (LIVE) return await A.LinkWithPhone(phone); return ""; }
// Timer hilang-otomatis default (detik) untuk chat baru.
export function setDefaultDisappearing(seconds) { if (LIVE) A.SetDefaultDisappearing(seconds); }
// QR kontak sendiri (PNG data-URI; revoke=buat ulang).
export async function myQR(revoke = false) { if (LIVE) return await A.MyQR(revoke); return ""; }

// Unduh media dari URL web (sisi Go → tanpa CORS) → data-URI. "" bila gagal.
export async function fetchRemoteMedia(url) {
  if (LIVE) return A.FetchRemoteMedia(url);
  return "";
}

// Pratinjau tautan (OG meta, diambil sisi Go → tanpa CORS). Cache per-URL.
const _lpCache = {};
export async function getLinkPreview(url) {
  if (!LIVE || !url) return null;
  if (url in _lpCache) return _lpCache[url];
  let p = null;
  try { p = await A.GetLinkPreview(url); } catch (e) {}
  _lpCache[url] = p;
  return p;
}

// Langganan event dari engine (Wails runtime global). Kembalikan fungsi
// unsubscribe (panggil di onDestroy) agar listener tak menumpuk per-mount.
export function onEvent(name, cb) {
  if (typeof window !== "undefined" && window.runtime?.EventsOn) {
    const off = window.runtime.EventsOn(name, cb);
    return typeof off === "function" ? off : () => window.runtime.EventsOff?.(name);
  }
  return () => {};
}
