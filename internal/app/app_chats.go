package app

// app_chats.go — data daftar chat (sidebar) & pesan satu percakapan untuk
// frontend, plus DTO JSON-nya. Penamaan & urutan di-resolve di sini.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"whatsapp-lite/internal/storage"
)

// mentionRe cocok dgn @<nomor> (mention WhatsApp dalam teks).
var mentionRe = regexp.MustCompile(`@(\d{6,})`)

// resolveMentions mengganti "@<nomor>" dgn "@<nama>" (utk preview sidebar / kutipan).
func (a *App) resolveMentions(text string) string {
	if !strings.Contains(text, "@") {
		return text
	}
	return mentionRe.ReplaceAllStringFunc(text, func(tok string) string {
		if n, _ := a.mentionName(tok[1:]); n != "" {
			return "@" + n
		}
		return tok
	})
}

// mentionName: resolve nomor mention → (nama, jid). Selalu kembalikan jid yg
// terpetakan; nama fallback ke nomor ("+62…") bila tak dikenal — tetap klikable.
func (a *App) mentionName(num string) (string, string) {
	// Pilih bentuk jid yg punya nama; kalau tak ada, default ke @s.whatsapp.net.
	jid := num + "@s.whatsapp.net"
	for _, suf := range []string{"@s.whatsapp.net", "@lid", "@bot"} {
		cand := num + suf
		if n, _ := a.nameOf(cand); n != "" {
			return n, cand
		}
	}
	if p := a.phoneOf(jid); p != "" {
		return p, jid
	}
	return "+" + num, jid
}

// MentionDTO = satu mention dalam teks (utk FE render berwarna + klik→profil).
type MentionDTO struct {
	Num  string `json:"num"`  // angka setelah @ (token di teks)
	Name string `json:"name"` // nama tampil (fallback "+nomor")
	JID  string `json:"jid"`  // jid utk buka profil/chat
}

// buildMentions mengumpulkan SEMUA @<nomor> dalam teks (termasuk yg tak dikenal,
// mis. Meta AI → tampil "+nomor" tapi tetap klikable ke profil).
func (a *App) buildMentions(text string) []MentionDTO {
	if !strings.Contains(text, "@") {
		return nil
	}
	var out []MentionDTO
	seen := map[string]bool{}
	for _, m := range mentionRe.FindAllStringSubmatch(text, -1) {
		num := m[1]
		if seen[num] {
			continue
		}
		seen[num] = true
		name, jid := a.mentionName(num)
		out = append(out, MentionDTO{Num: num, Name: name, JID: jid})
	}
	return out
}

// --- DTO JSON untuk frontend ---

type ChatDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Preview string `json:"preview"`
	Time    string `json:"time"`
	Ts      int64  `json:"ts"`
	Group   bool   `json:"group"`
	Sent    bool   `json:"sent"`       // pesan terakhir dari kita → tampilkan centang
	Status  string `json:"status"`     // status pesan keluar terakhir (sent/delivered/read)
	Unread  bool   `json:"unread"`
	Badge   int    `json:"badge"`
	Pinned  bool   `json:"pinned"`
	Muted   bool   `json:"muted"`
}

func (a *App) GetChats() (out []ChatDTO) {
	out = []ChatDTO{}
	defer func() {
		if r := recover(); r != nil {
			out = []ChatDTO{}
			runtime.LogError(a.ctx, fmt.Sprintf("GetChats recover: %v", r))
		}
	}()
	if a.store == nil {
		return
	}
	// RecomputeSummaries TIDAK lagi di sini (dulu O(chats) tiap refresh).
	// Ringkasan di-update incremental saat SaveMessage; recompute sekali di startup.
	cs, err := a.store.ListChats(a.ctx)
	if err != nil {
		return
	}
	for _, c := range cs {
		if c.JID == "status@broadcast" {
			continue // bukan chat; jangan tampil di daftar
		}
		out = append(out, a.chatDTO(c))
	}
	return out
}

// GetArchivedChats mengembalikan chat yang diarsipkan (panel terpisah).
func (a *App) GetArchivedChats() (out []ChatDTO) {
	out = []ChatDTO{}
	if a.store == nil {
		return
	}
	cs, err := a.store.ListArchivedChats(a.ctx)
	if err != nil {
		return
	}
	for _, c := range cs {
		if c.JID == "status@broadcast" {
			continue
		}
		out = append(out, a.chatDTO(c))
	}
	return out
}

// chatDTO memetakan satu chat storage → DTO (resolusi nama + preview grup).
func (a *App) chatDTO(c storage.Chat) ChatDTO {
	// Prioritas: nama kontak tersimpan / subjek grup (otoritatif) >
	// nama tersimpan di DB (pushname) > nomor terbaca > short JID.
	// Grup: pakai subjek ter-cache di DB dulu → hindari GetGroupInfo (jaringan)
	// tiap refresh sidebar. 1:1: nama via nameOf (label/buku-alamat/pushname).
	name := ""
	if isGroupJID(c.JID) && c.Name != "" {
		name = c.Name
	} else {
		name, _ = a.nameOf(c.JID)
		if name == "" {
			name = c.Name
		}
	}
	if name == "" {
		name = a.phoneOf(c.JID)
	}
	if name == "" {
		name = shortJID(c.JID)
	}
	// Preview grup: prefix "Nama: " (atau "Kamu: ") seperti WhatsApp.
	preview := c.LastText
	if isGroupJID(c.JID) && preview != "" {
		if c.LastFromMe {
			preview = "Kamu: " + preview
		} else if c.LastSender != "" {
			preview = c.LastSender + ": " + preview
		}
	}
	status := ""
	if c.LastFromMe {
		status = c.LastStatus
		if status == "" {
			status = "sent"
		}
	}
	return ChatDTO{
		ID: c.JID, Name: name, Preview: preview,
		Time: relTime(c.LastTS), Ts: c.LastTS.Unix(), Group: isGroupJID(c.JID),
		Sent: c.LastFromMe, Status: status,
		Unread: c.Unread > 0, Badge: c.Unread, Pinned: c.Pinned, Muted: c.Muted,
	}
}

// ExportChat membuat transkrip teks polos seluruh riwayat chat (utk diunduh).
func (a *App) ExportChat(jid string) string {
	if a.store == nil {
		return ""
	}
	ms, err := a.store.ListMessages(a.ctx, jid, 100000)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, m := range ms {
		name := "Saya"
		if !m.FromMe {
			if name, _ = a.nameOf(m.Sender); name == "" {
				name = m.PushName
			}
			if name == "" {
				name = shortJID(m.Sender)
			}
		}
		text := m.Text
		if text == "" {
			text = "<" + nonEmpty(m.Kind, "media") + ">"
		}
		b.WriteString(m.Timestamp.Format("2006-01-02 15:04"))
		b.WriteString(" - ")
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(text)
		b.WriteByte('\n')
	}
	return b.String()
}

type MessageDTO struct {
	ID       string `json:"id"`
	Dir      string `json:"dir"`
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thumb    string `json:"thumb"`
	Time     string `json:"time"`
	Sender      string `json:"sender"`      // nama tampil pengirim (grup)
	SenderID    string `json:"senderId"`    // jid pengirim (utk foto/profil)
	SenderPhone string `json:"senderPhone"` // "+62…" (grup, tak-tersimpan)
	SenderSaved bool   `json:"senderSaved"` // pengirim ada di label/buku-alamat
	Status      string `json:"status"`
	Pinned    bool         `json:"pinned"` // disematkan di chat
	Edited    bool         `json:"edited"` // pernah disunting
	Ts        int64        `json:"ts"` // unix detik (kursor pagination)
	QuoteID   string       `json:"quoteId"`   // balasan: id pesan dikutip (utk lompat)
	QuoteName string       `json:"quoteName"` // balasan: nama pengirim yg dikutip
	QuoteText string       `json:"quoteText"` // balasan: preview teks dikutip
	Mentions  []MentionDTO  `json:"mentions"`  // @tag dlm teks (render berwarna+klik)
	Reactions []ReactionDTO `json:"reactions"` // reaksi emoji teragregasi
}

// ReactionDTO = satu emoji teragregasi pada pesan.
type ReactionDTO struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Mine  bool   `json:"mine"`
}

// attachReactions mengisi field Reactions tiap DTO dari peta reaksi chat.
func (a *App) attachReactions(out []MessageDTO, chat string) {
	if a.store == nil || len(out) == 0 {
		return
	}
	rmap, err := a.store.ReactionsForChat(a.ctx, chat)
	if err != nil || len(rmap) == 0 {
		return
	}
	self := ""
	if a.eng != nil {
		self = userPart(a.eng.SelfJID())
	}
	for i := range out {
		rs := rmap[out[i].ID]
		if len(rs) == 0 {
			continue
		}
		order := []string{}
		agg := map[string]*ReactionDTO{}
		for _, r := range rs {
			d := agg[r.Emoji]
			if d == nil {
				d = &ReactionDTO{Emoji: r.Emoji}
				agg[r.Emoji] = d
				order = append(order, r.Emoji)
			}
			d.Count++
			if self != "" && userPart(r.Sender) == self {
				d.Mine = true
			}
		}
		for _, e := range order {
			out[i].Reactions = append(out[i].Reactions, *agg[e])
		}
	}
}

// GetMessages: 200 pesan terbaru.
func (a *App) GetMessages(jid string) (out []MessageDTO) {
	out = []MessageDTO{}
	defer func() {
		if r := recover(); r != nil {
			out = []MessageDTO{}
			runtime.LogError(a.ctx, fmt.Sprintf("GetMessages recover: %v", r))
		}
	}()
	if a.store == nil {
		return
	}
	ms, err := a.store.ListMessages(a.ctx, jid, 200)
	if err != nil {
		return
	}
	out = a.toDTO(ms)
	a.attachReactions(out, jid)
	return
}

// GetMessagesBefore: hingga 50 pesan lebih LAMA dari beforeTs (pagination scroll atas).
func (a *App) GetMessagesBefore(jid string, beforeTs int64) (out []MessageDTO) {
	out = []MessageDTO{}
	defer func() {
		if r := recover(); r != nil {
			out = []MessageDTO{}
		}
	}()
	if a.store == nil {
		return
	}
	ms, err := a.store.ListMessagesBefore(a.ctx, jid, beforeTs, 50)
	if err != nil {
		return
	}
	out = a.toDTO(ms)
	a.attachReactions(out, jid)
	return
}

// toDTO memetakan pesan storage → DTO frontend (nama, balasan, mention).
func (a *App) toDTO(ms []storage.Message) []MessageDTO {
	out := []MessageDTO{}
	// Memo per-pengirim: grup punya sedikit pengirim unik tapi ratusan pesan.
	// Tanpa memo, resolusi nama/nomor (label + buku-alamat + lid-lookup) jalan
	// per pesan × 200 → buka chat lambat/seret. Memo → sekali per pengirim.
	type sInfo struct {
		name  string
		saved bool
		phone string
	}
	memo := map[string]sInfo{}
	resolveSender := func(jid, rowPush string) sInfo {
		if v, ok := memo[jid]; ok {
			return v
		}
		// label lokal > buku-alamat > pushname baris (TANPA query DB per pesan).
		name, saved := "", false
		if lbl := a.labelOf(jid); lbl != "" {
			name, saved = lbl, true
		} else if a.eng != nil {
			name, saved = a.eng.ResolveName(jid)
		}
		if name == "" {
			name = rowPush // pushname pesan ini (sudah termuat)
		}
		phone := ""
		if !saved && jid != "" {
			phone = a.phoneOf(jid)
		}
		if name == "" {
			if phone != "" {
				name = phone
			} else {
				name = shortJID(jid)
			}
		}
		v := sInfo{name, saved, phone}
		memo[jid] = v
		return v
	}
	for _, m := range ms {
		dir := "in"
		if m.FromMe {
			dir = "out"
		}
		kind := m.Kind
		if kind == "" {
			kind = "text"
		}
		// Nama pengirim: label lokal > buku-alamat > pushname (memo). saved=false
		// → grup tampil "Nama + nomor".
		si := resolveSender(m.Sender, m.PushName)
		senderName, senderSaved, senderPhone := si.name, si.saved, si.phone
		quoteName := ""
		if m.QuotedText != "" || m.QuotedID != "" {
			if m.QuotedSender != "" {
				quoteName = resolveSender(m.QuotedSender, "").name
			}
			if quoteName == "" {
				quoteName = "Kamu"
			}
		}
		status := ""
		if m.FromMe {
			status = m.Status
			if status == "" {
				status = "sent"
			}
		}
		out = append(out, MessageDTO{
			ID: m.ID, Dir: dir, Type: kind, Text: m.Text, Thumb: m.Thumb,
			Time: hm(m.Timestamp), Ts: m.Timestamp.Unix(), Sender: senderName, SenderID: m.Sender,
			SenderPhone: senderPhone, SenderSaved: senderSaved, Status: status,
			Pinned:    m.Pinned, Edited: m.Edited,
			QuoteID: m.QuotedID, QuoteName: quoteName, QuoteText: a.resolveMentions(m.QuotedText),
			Mentions: a.buildMentions(m.Text),
		})
	}
	return out
}
