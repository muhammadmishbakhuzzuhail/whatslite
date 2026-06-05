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
  if (m.quoteText || m.quoteName) o.quote = { name: m.quoteName || "", text: m.quoteText || "" };
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
export async function sendMedia(jid, kind, caption, fileName, dataURI) {
  if (LIVE) return A.SendMedia(jid, kind, caption, fileName, dataURI);
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
export async function searchMessages(query) {
  if (LIVE) return (await A.SearchMessages(query)) || [];
  return [];
}
export async function getStarred() {
  if (LIVE) return (await A.GetStarred()) || [];
  return [];
}

// --- Status / Stories ---
export async function getStatuses() {
  if (LIVE) return (await A.GetStatuses()) || [];
  return [];
}
export async function postTextStatus(text) {
  if (LIVE) return A.PostTextStatus(text);
  return "";
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
export function unfollowChannel(jid) {
  if (LIVE) A.UnfollowChannel(jid);
}
export function muteChannel(jid, on) {
  if (LIVE) A.MuteChannel(jid, on);
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
export function block(jid, on) {
  if (LIVE) A.Block(jid, on);
}
export function setMyName(name) {
  if (LIVE) A.SetMyName(name);
}
export function setMyAbout(text) {
  if (LIVE) A.SetMyAbout(text);
}

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

// Notifikasi desktop native (Linux). No-op di browser/preview.
export function notify(title, body) {
  if (LIVE) A.Notify(title, body);
}

// Langganan event dari engine (Wails runtime global).
export function onEvent(name, cb) {
  if (typeof window !== "undefined" && window.runtime?.EventsOn) {
    window.runtime.EventsOn(name, cb);
  }
}
