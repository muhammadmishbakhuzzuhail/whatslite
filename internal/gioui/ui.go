// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// ui.go — tata letak 3-panel (rail | sidebar | percakapan), daftar chat, bubble
// pesan, avatar. Menggambar bentuk kustom (RRect bubble, lingkaran avatar) via
// clip — membuktikan Gio bisa desain pixel-WhatsApp. Data dari engine in-process.
package gioui

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/richtext"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
)

type UI struct {
	th         *material.Theme
	core       *app.App
	t          Theme
	dark       bool
	state      string
	qr         string // kode QR pairing mentah terbaru (dari core.QRCode); "" = belum ada
	subtitle   string // subtitle header chat (online/mengetik…/terakhir dilihat)
	typingSent bool   // status composing terakhir yg dikirim (throttle SendTyping)
	view       string // pane sidebar aktif: chats|calls|settings

	olderReqChat string    // chat terakhir diminta history lama (throttle pagination)
	olderReqAt   time.Time // waktu permintaan history lama terakhir

	pollClicks    map[string][]widget.Clickable // msgID → clickable per opsi polling
	pollVoteCache map[string]pollVoteEntry      // msgID → hasil suara (TTL — hindari query/frame)
	mentionState  richtext.InteractiveText      // state teks ber-mention (warna inline)

	statusGroupsCache []app.StatusGroupDTO // grup status terkini (utk viewer)
	statusClicks      []widget.Clickable
	statusViewIdx     int
	statusClose       widget.Clickable

	contactFlat       []app.ContactRowDTO // kontak datar (pane Kontak → buka chat)
	contactPaneClicks []widget.Clickable

	// buat grup (Kontak → "Grup baru"): nama + multi-pilih kontak.
	gcNewBtn widget.Clickable
	gcNameEd widget.Editor
	gcSel    map[string]bool // jid → terpilih
	gcClicks []widget.Clickable
	gcCreate widget.Clickable
	gcCancel widget.Clickable
	gcList   widget.List

	// cache TTL pembangun data pane (hindari query DB tiap frame saat scroll pane).
	cgCache                []cpGroup
	srCache                []stpItem
	crCache                []spCall
	chCache                []chnChannel
	cgAt, srAt, crAt, chAt time.Time

	// alur login via nomor telepon (alternatif QR): toggle, input, kode 8-karakter.
	loginPhone  bool
	phoneEd     widget.Editor
	loginSwitch widget.Clickable
	loginSubmit widget.Clickable
	pairCode    string

	setClicks [8]widget.Clickable // baris pane setelan (0=Tema … 7=Keluar)

	// pencarian + filter daftar chat (paritas SearchBar.svelte + Filters.svelte).
	searchEd     widget.Editor
	filterSel    int // 0 Semua · 1 Belum dibaca · 2 Favorit · 3 Grup
	filterClicks [4]widget.Clickable
	shown        []int            // indeks u.chats yg lolos filter+pencarian (urut tampil)
	newChatClick widget.Clickable // baris "mulai chat baru" (query nomor)

	// mode balas: pesan yg dikutip; "" = kirim biasa.
	replyTo   string
	replyName string
	replyText string
	replyX    widget.Clickable // tombol batal balas

	// pemilih reaksi: target pesan (kosong = mode sisip emoji ke editor).
	rpClicks    []widget.Clickable
	reactMsgID  string
	reactSender string
	reactFromMe bool

	// teruskan: id pesan sumber + klik per-chat tujuan + batal.
	fwdMsgID  string
	fwdClicks []widget.Clickable
	fwdCancel widget.Clickable

	// menu lampiran (tombol "+"): klik per-baris.
	attachClicks []widget.Clickable

	// penyusun polling (lampiran → Polling): pertanyaan + opsi + tombol.
	pollQEd    widget.Editor
	pollOptEds [4]widget.Editor
	pollCreate widget.Clickable
	pollCancel widget.Clickable

	// kirim kontak (lampiran → Kontak): daftar kontak + pilih.
	contactSendCache  []app.ContactRowDTO
	contactSendClicks []widget.Clickable
	contactSendCancel widget.Clickable
	contactSendList   widget.List

	// kirim lokasi (lampiran → Lokasi): nama tempat + lat/lng.
	locNameEd widget.Editor
	locLatEd  widget.Editor
	locLngEd  widget.Editor
	locSend   widget.Clickable
	locCancel widget.Clickable

	// menu aksi baris chat (klik-kanan): SNAPSHOT chat saat menu dibuka (aksi pakai
	// ini, bukan index — u.chats di-replace tiap refresh & bisa reorder).
	chatCtxChat   app.ChatDTO
	chatCtxItems  [5]widget.Clickable
	headMenuClick widget.Clickable // ikon overflow header → menu chat terbuka

	// sub-pane setelan (profil/penyimpanan) + navigasi kembali.
	setSub          string
	setBack         widget.Clickable
	setProfileClick widget.Clickable
	profNameEd      widget.Editor // edit nama (sub-pane profil)
	profAboutEd     widget.Editor // edit tentang
	profSave        widget.Clickable
	profLoaded      bool                // editor sudah diisi nilai saat ini?
	privacyClicks   [8]widget.Clickable // baris privasi (siklus nilai → SetPrivacy)

	chats     []app.ChatDTO
	selected  string
	selName   string
	selGroup  bool
	messages  []app.MessageDTO
	lastFetch time.Time

	chatList   widget.List
	msgList    widget.List
	clicks     []widget.Clickable
	railClicks []widget.Clickable
	editor     widget.Editor
	photos     map[string]paint.ImageOp // foto avatar in-memory (nama → op)
	photoMu    sync.Mutex               // lindungi photos (diisi dari goroutine loader)
	photoTried map[string]bool          // jid yg sudah dicoba ambil (hindari refetch)

	media      map[string]paint.ImageOp // thumbnail media bubble (msgID → op)
	mediaMu    sync.Mutex
	mediaTried map[string]bool // msgID yg sudah dicoba ambil

	overlay     string // popup aktif: ""|info|reaction|forward|msginfo|picker|lightbox|msgctx
	headerClick widget.Clickable
	emojiClick  widget.Clickable
	attachClick widget.Clickable
	backdrop    widget.Clickable
	msgClicks   []widget.Clickable
	ctxIdx      int            // index pesan utk context-menu (display only)
	ctxMsg      app.MessageDTO // SNAPSHOT pesan saat menu dibuka — aksi pakai ini, bukan
	// index: backfill history prepend & refresh reorder menggeser semua index.
	ctxItems [6]widget.Clickable // item menu (react/reply/forward/star/info/delete)

	// OnPlayVoice/OnPlayVideo: hook media (di-set cmd/whatslite-gio → internal/
	// voice + internal/video). gioui TETAP bebas-cgo (gio-shot ringan).
	OnPlayVoice func(chat, id string)
	OnPlayVideo func(chat, id, typ string)
	// OnAttach: hook pilih-berkas + kirim (di-set cmd/whatslite-gio → x/explorer +
	// core.SendMedia). category ∈ media|document|contact|location|poll. Pisah dari
	// gioui agar tetap bebas-window/cgo.
	OnAttach func(chat, category string)
}

// ctxMenu = item context-menu pesan (glyph + aksi/overlay tujuan).
var ctxMenu = []struct{ icon, label, to string }{
	{"emoji", "Reaksi", "reaction"}, {"reply", "Balas", ""}, {"forward", "Teruskan", "forward"},
	{"star", "Bintangi", ""}, {"info", "Info", "msginfo"}, {"trash", "Hapus", ""},
}

// SetOverlay: utk render-tool menguji popup headless.
func (u *UI) SetOverlay(o string) { u.overlay = o }

// SetReply: utk render-tool menguji banner balas headless.
func (u *UI) SetReply(name, text string) { u.replyTo, u.replyName, u.replyText = "demo", name, text }

// ScrollMessagesToEnd: utk render-tool menguji gulir-ke-bawah headless.
func (u *UI) ScrollMessagesToEnd() { u.msgList.ScrollTo(1 << 20) }

// SetSearch: utk render-tool menguji bilah cari / tawaran chat-baru headless.
func (u *UI) SetSearch(s string) { u.searchEd.SetText(s) }

// SetSettingsSub: utk render-tool menguji sub-pane setelan headless.
func (u *UI) SetSettingsSub(s string) { u.view = "settings"; u.setSub = s }

// railNav = tombol nav rail kiri (ikon SVG WhatsApp + view tujuan).
var railNav = []struct{ view, icon string }{
	{"chats", "chats"}, {"status", "status"}, {"channels", "channels"},
	{"calls", "calls"}, {"contacts", "contacts"}, {"settings", "settings"},
}

func NewUI(th *material.Theme, core *app.App) *UI {
	u := &UI{th: th, core: core, dark: true, view: "chats"}
	u.t = newTheme(u.dark)
	u.chatList.Axis = layout.Vertical
	u.msgList.Axis = layout.Vertical
	u.railClicks = make([]widget.Clickable, len(railNav))
	u.editor.SingleLine = true
	u.editor.Submit = true
	u.phoneEd.SingleLine = true
	u.phoneEd.Submit = true
	u.searchEd.SingleLine = true
	u.profNameEd.SingleLine = true
	u.profAboutEd.SingleLine = true
	u.pollQEd.SingleLine = true
	for i := range u.pollOptEds {
		u.pollOptEds[i].SingleLine = true
	}
	u.locNameEd.SingleLine = true
	u.locLatEd.SingleLine = true
	u.locLngEd.SingleLine = true
	u.gcNameEd.SingleLine = true
	u.gcSel = map[string]bool{}
	u.rpClicks = make([]widget.Clickable, len(RpEmoji()))
	u.photos = map[string]paint.ImageOp{}
	u.photoTried = map[string]bool{}
	u.media = map[string]paint.ImageOp{}
	u.mediaTried = map[string]bool{}
	u.pollClicks = map[string][]widget.Clickable{}
	u.pollVoteCache = map[string]pollVoteEntry{}
	if core == nil { // demo: foto sintetis utk membuktikan avatar-foto bulat + thumb
		u.photos["Andi Pratama"] = synthPhoto()
		u.media["m13"] = synthPhoto() // bubble image demo (m14 video = placeholder+play)
	}
	return u
}

// SetDark: ganti tema (dipakai render-tool utk audit light/dark).
func (u *UI) SetDark(d bool) { u.dark = d; u.t = newTheme(d) }

// SetView/Deselect: utk render-tool menguji state navigasi headless.
func (u *UI) SetView(v string) { u.view = v }
func (u *UI) Deselect()        { u.selected = "" }

// View/Overlay — getter agar render-tool bisa simpan+pulihkan state saat memotret
// layar bernama dari app yg sedang berjalan (WLGIO_SHOT_SCREENS).
func (u *UI) View() string    { return u.view }
func (u *UI) Overlay() string { return u.overlay }

func (u *UI) refresh() {
	if u.core == nil { // mode demo: data statis (uji render tanpa engine/jaringan)
		u.chats = demoChats()
		if u.selected == "" {
			u.selected, u.selName, u.selGroup = "2", "Keluarga", true
		}
		u.messages = demoMessages()
		u.subtitle = "online"
	} else {
		u.state = u.core.GetState()
		u.qr = u.core.QRCode()
		u.chats = u.core.GetChats()
		if u.selected != "" {
			u.messages = u.core.GetMessages(u.selected)
			u.subtitle = u.core.ChatSubtitle(u.selected)
		}
	}
	if len(u.clicks) < len(u.chats) {
		u.clicks = make([]widget.Clickable, len(u.chats))
	}
	if len(u.msgClicks) < len(u.messages) {
		u.msgClicks = make([]widget.Clickable, len(u.messages))
	}
}

func demoChats() []app.ChatDTO {
	return []app.ChatDTO{
		{ID: "1", Name: "Andi Pratama", Preview: "Mantap! Sampai nanti malam 🙌", Time: "19.08", Sent: true, Status: "read", Pinned: true},
		{ID: "2", Name: "Keluarga", Preview: "Ibu: Jangan lupa makan ya nak", Time: "18.41", Group: true, Badge: 2, Unread: true},
		{ID: "3", Name: "Sarah", Preview: "Oke besok aku kabarin lagi", Time: "17.55", Sent: true, Status: "sent"},
		{ID: "4", Name: "Tim Proyek X", Preview: "Budi: file-nya udah aku upload", Time: "16.20", Group: true, Badge: 12, Unread: true},
		{ID: "5", Name: "Rian", Preview: "Haha iya bener banget 😄", Time: "14.03", Muted: true},
	}
}
func demoMessages() []app.MessageDTO {
	now := time.Now().Unix()
	yest := now - 86400
	return []app.MessageDTO{
		{ID: "m1", Dir: "in", Type: "text", Text: "Halo! Jadi nanti malam ngumpul jam berapa?", Time: "19.02", Sender: "Budi Santoso", Ts: yest},
		{ID: "m2", Dir: "out", Type: "text", Text: "Jam 8 ya, di tempat biasa 👍", Time: "19.03", Status: "delivered", Ts: yest},
		{ID: "m3", Dir: "in", Type: "text", Text: "Sip. Tempatnya yang kemarin kan?", Time: "19.05", Sender: "Budi Santoso", Ts: yest},
		{ID: "m4", Dir: "out", Type: "text", Text: "Iya betul, yang deket stasiun", Time: "19.06", Status: "read", Ts: yest, QuoteName: "Budi Santoso", QuoteText: "Sip. Tempatnya yang kemarin kan?", Reactions: []app.ReactionDTO{{Emoji: "👍", Count: 2}, {Emoji: "🔥", Count: 1, Mine: true}}},
		{ID: "m5", Dir: "in", Type: "text", Text: "Aku mungkin telat dikit, macet", Time: "19.40", Sender: "Citra Dewi", Ts: yest},
		{ID: "m6", Dir: "out", Type: "text", Text: "Santai, kita tunggu", Time: "19.41", Status: "read", Ts: yest, Edited: true},
		{ID: "m6b", Dir: "in", Type: "text", Time: "19.42", Sender: "Citra Dewi", Ts: yest, Revoked: true},
		{ID: "m7", Dir: "in", Type: "text", Text: "Oke sip. Aku bawa kamera sekalian foto-foto.", Time: "08.04", Sender: "Citra Dewi", Ts: now},
		{ID: "m8", Dir: "out", Type: "text", Text: "Mantap! Jangan lupa baterai cadangan", Time: "08.05", Status: "read", Ts: now},
		{ID: "m9", Dir: "in", Type: "text", Text: "Udah siap semua kok 📸", Time: "08.06", Sender: "Citra Dewi", Ts: now},
		{ID: "m10", Dir: "in", Type: "text", Text: "Btw jadi makan dulu apa langsung?", Time: "08.07", Sender: "Budi Santoso", Ts: now},
		{ID: "m11", Dir: "out", Type: "text", Text: "Makan dulu aja, laper nih 😅", Time: "08.08", Status: "delivered", Ts: now},
		{ID: "m12", Dir: "out", Type: "text", Text: "Mantap! Sampai nanti 🙌", Time: "08.09", Status: "sent", Ts: now},
		{ID: "m13", Dir: "in", Type: "image", Text: "Spot foto kemarin 📷", Time: "08.10", Sender: "Citra Dewi", Ts: now},
		{ID: "m14", Dir: "out", Type: "video", Time: "08.11", Status: "delivered", Ts: now},
		{ID: "m15", Dir: "in", Type: "poll", Text: "Liburan ke mana minggu depan?", Thumb: `["Pantai","Gunung","Kota"]`, Time: "08.12", Sender: "Budi Santoso", Ts: now},
		{ID: "m16", Dir: "in", Type: "document", Text: "Laporan_Tahunan_2026.pdf", DocMime: "application/pdf", DocSize: 524288, DocPages: 12, Time: "08.13", Sender: "Citra Dewi", Ts: now},
		{ID: "m17", Dir: "out", Type: "voice", Text: "0:12", Time: "08.14", Status: "read", Ts: now},
		{ID: "m18", Dir: "in", Type: "location", Text: "Jl. Sudirman No. 12, Jakarta", Time: "08.15", Sender: "Budi Santoso", Ts: now},
		{ID: "m19", Dir: "in", Type: "contact", Text: "Dewi Anggraini", Thumb: "+62 812-3456-7890", Time: "08.16", Sender: "Citra Dewi", Ts: now},
		{ID: "m20", Dir: "in", Type: "text", Text: "Setuju sama @Budi Santoso, nanti @Citra Dewi yang bawa kamera ya", Time: "08.17", Sender: "Rian", Ts: now, Mentions: []app.MentionDTO{{Name: "Budi Santoso"}, {Name: "Citra Dewi"}}},
	}
}

func (u *UI) Layout(gtx layout.Context) layout.Dimensions {
	if time.Since(u.lastFetch) > 600*time.Millisecond {
		u.refresh()
		u.lastFetch = time.Now()
	}
	// latar
	paint.FillShape(gtx.Ops, u.t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// Gerbang login: engine tersambung tapi sesi belum siap → layar QR / nomor.
	if u.core != nil && u.state != "" && u.state != "ready" && u.state != "connected" {
		u.handleLogin(gtx)
		return LoginView(gtx, u.th, u.t, u.qr, u.loginCtl())
	}

	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.rail(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.sidebar(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.conversation(gtx) }),
	)
	if u.overlay != "" {
		u.overlayLayer(gtx)
	}
	return dims
}

// handleSettings memproses klik baris pane setelan: Tema (toggle gelap/terang)
// dan Keluar (logout engine → kembali ke layar QR).
func (u *UI) handleSettings(gtx layout.Context) {
	for u.setBack.Clicked(gtx) { // kembali dari sub-pane
		u.setSub = ""
		u.profLoaded = false
	}
	for u.setProfileClick.Clicked(gtx) { // kartu profil → sub-pane profil
		u.setSub = "profile"
		u.profLoaded = false
	}
	for u.profSave.Clicked(gtx) { // simpan nama/tentang
		if u.core != nil {
			u.core.SetMyName(strings.TrimSpace(u.profNameEd.Text()))
			u.core.SetMyAbout(strings.TrimSpace(u.profAboutEd.Text()))
		}
	}
	for u.setClicks[0].Clicked(gtx) { // Tema
		u.dark = !u.dark
		u.t = newTheme(u.dark)
	}
	for u.setClicks[3].Clicked(gtx) { // Simpan pesan dihapus (anti-delete)
		if u.core != nil {
			u.core.SetKeepDeleted(!u.core.GetKeepDeleted())
		}
	}
	for u.setClicks[5].Clicked(gtx) { // Privasi → sub-pane
		u.setSub = "privacy"
	}
	for u.setClicks[6].Clicked(gtx) { // Penyimpanan → sub-pane
		u.setSub = "storage"
	}
	for u.setClicks[7].Clicked(gtx) { // Keluar
		if u.core != nil {
			u.core.Logout()
			u.state = "qr"
		}
	}
}

// loginCtl membangun controller login dari state UI (utk LoginView interaktif).
func (u *UI) loginCtl() *LoginCtl {
	return &LoginCtl{
		PhoneMode: u.loginPhone, Phone: &u.phoneEd,
		Switch: &u.loginSwitch, Submit: &u.loginSubmit, Code: u.pairCode,
	}
}

// handleLogin memproses event layar login: toggle QR↔nomor + minta kode pairing.
func (u *UI) handleLogin(gtx layout.Context) {
	for u.loginSwitch.Clicked(gtx) {
		u.loginPhone = !u.loginPhone
		u.pairCode = ""
	}
	req := false
	for u.loginSubmit.Clicked(gtx) {
		req = true
	}
	for {
		ev, ok := u.phoneEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			req = true
		}
	}
	if req && u.core != nil {
		if phone := strings.TrimSpace(u.phoneEd.Text()); phone != "" {
			u.pairCode = u.core.LinkWithPhone(phone)
		}
	}
}

// overlayLayer — popup di atas app (backdrop klik → tutup). Komponen wave dipakai
// langsung; modal punya backdrop sendiri, info-drawer di-posisikan kanan.
func (u *UI) overlayLayer(gtx layout.Context) {
	for u.backdrop.Clicked(gtx) {
		u.overlay = ""
	}
	// backdrop clickable penuh (di belakang isi)
	u.backdrop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	switch u.overlay {
	case "info":
		// drawer kanan 400px + dim di kiri-nya
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		w := gtx.Dp(400)
		off := op.Offset(image.Pt(gtx.Constraints.Max.X-w, 0)).Push(gtx.Ops)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		InfoDrawerView(gtx, u.th, u.t, u.infoData())
		off.Pop()
	case "forward":
		u.handleForward(gtx)
		ModalsView(gtx, u.th, u.t, u.fwdCtl())
	case "msginfo":
		MsgInfoView(gtx, u.th, u.t)
	case "reaction":
		u.handleReaction(gtx)
		paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X, gtx.Constraints.Max.Y = gtx.Dp(352), gtx.Dp(400)
			return ReactionPickerView(gtx, u.th, u.t, &RpCtl{Clicks: u.rpClicks})
		})
	case "lightbox":
		LightboxView(gtx, u.th, u.t)
	case "picker":
		PickerView(gtx, u.th, u.t)
	case "attach":
		u.handleAttach(gtx)
		if len(u.attachClicks) < AttachCount() {
			u.attachClicks = make([]widget.Clickable, AttachCount())
		}
		AttachMenuView(gtx, u.th, u.t, &AttachCtl{Clicks: u.attachClicks})
	case "msgctx":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(220), gtx.Dp(220)
			return u.ctxMenuView(gtx)
		})
	case "chatctx":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(240), gtx.Dp(240)
			return u.chatCtxView(gtx)
		})
	case "statusview":
		u.statusViewLayer(gtx)
	case "pollcompose":
		u.pollComposeLayer(gtx)
	case "contactsend":
		u.contactSendLayer(gtx)
	case "loccompose":
		u.locComposeLayer(gtx)
	case "groupcreate":
		u.groupCreateLayer(gtx)
	}
}

// doCtxAction menjalankan aksi context-menu pesan terhadap engine. Bintangi/Hapus
// langsung; Balas mengaktifkan banner balas di composer (kirim → core.Reply).
func (u *UI) doCtxAction(label string) {
	m := u.ctxMsg // snapshot saat menu dibuka (bukan index yg bisa bergeser)
	if m.ID == "" {
		return
	}
	fromMe := m.Dir == "out"
	switch label {
	case "Bintangi":
		if u.core != nil {
			u.core.StarMessage(u.selected, m.ID, m.SenderID, fromMe, true)
		}
	case "Hapus":
		if u.core != nil {
			u.core.DeleteMessage(u.selected, m.ID, m.SenderID, fromMe, fromMe) // everyone hanya utk pesan sendiri
			u.messages = u.core.GetMessages(u.selected)
		}
	case "Balas":
		u.replyTo, u.replyName, u.replyText = m.ID, u.replyDisplayName(m), m.Text
	case "Reaksi":
		u.reactMsgID, u.reactSender, u.reactFromMe = m.ID, m.SenderID, fromMe // target reaksi
	case "Teruskan":
		u.fwdMsgID = m.ID // pesan sumber utk diteruskan (sumber = u.selected)
	}
}

// handleForward memproses modal teruskan: ketuk chat tujuan → core.Forward; batal.
func (u *UI) handleForward(gtx layout.Context) {
	if len(u.fwdClicks) < len(u.chats) {
		u.fwdClicks = make([]widget.Clickable, len(u.chats))
	}
	for u.fwdCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for i := range u.chats {
		if i >= len(u.fwdClicks) {
			break
		}
		for u.fwdClicks[i].Clicked(gtx) {
			if u.core != nil && u.fwdMsgID != "" {
				u.core.Forward(u.selected, u.fwdMsgID, u.chats[i].ID)
			}
			u.fwdMsgID = ""
			u.overlay = ""
		}
	}
}

// handleAttach memproses klik menu lampiran → OnAttach(chat, kategori) lalu tutup.
func (u *UI) handleAttach(gtx layout.Context) {
	if len(u.attachClicks) < AttachCount() {
		u.attachClicks = make([]widget.Clickable, AttachCount())
	}
	for i := range u.attachClicks {
		for u.attachClicks[i].Clicked(gtx) {
			cat := AttachCategory(i)
			if cat == "poll" { // polling disusun in-app (input teks) → SendPoll
				u.overlay = "pollcompose"
				continue
			}
			if cat == "contact" { // pilih kontak → SendContact
				u.overlay = "contactsend"
				continue
			}
			if cat == "location" { // input tempat+koordinat → SendLocation
				u.overlay = "loccompose"
				continue
			}
			if u.OnAttach != nil && u.selected != "" {
				u.OnAttach(u.selected, cat) // media/document via dialog berkas; contact/location TODO
			}
			u.overlay = ""
		}
	}
}

// groupCreateLayer — modal buat grup: nama + daftar kontak multi-pilih + Buat.
func (u *UI) groupCreateLayer(gtx layout.Context) {
	if u.core != nil && u.contactSendCache == nil { // pakai cache kontak yg sama
		u.contactSendCache = u.core.GetContacts()
		sort.Slice(u.contactSendCache, func(i, j int) bool {
			return strings.ToLower(u.contactSendCache[i].Name) < strings.ToLower(u.contactSendCache[j].Name)
		})
	}
	if len(u.gcClicks) < len(u.contactSendCache) {
		u.gcClicks = make([]widget.Clickable, len(u.contactSendCache))
	}
	for u.gcCancel.Clicked(gtx) {
		u.overlay, u.contactSendCache = "", nil
		u.gcNameEd.SetText("")
		u.gcSel = map[string]bool{}
	}
	for i := range u.contactSendCache {
		if i >= len(u.gcClicks) {
			break
		}
		for u.gcClicks[i].Clicked(gtx) { // toggle pilih
			j := u.contactSendCache[i].JID
			u.gcSel[j] = !u.gcSel[j]
		}
	}
	for u.gcCreate.Clicked(gtx) {
		name := strings.TrimSpace(u.gcNameEd.Text())
		var members []string
		for _, c := range u.contactSendCache {
			if u.gcSel[c.JID] {
				members = append(members, c.JID)
			}
		}
		if name != "" && len(members) >= 1 && u.core != nil {
			if jid := u.core.CreateGroup(name, members); jid != "" {
				u.selected, u.selName, u.selGroup = jid, name, true
				u.view = "chats"
				u.core.OpenChat(jid)
				u.messages = u.core.GetMessages(jid)
			}
		}
		u.overlay, u.contactSendCache = "", nil
		u.gcNameEd.SetText("")
		u.gcSel = map[string]bool{}
	}
	u.gcList.Axis = layout.Vertical
	selCount := 0
	for _, v := range u.gcSel {
		if v {
			selCount++
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(390), gtx.Dp(520)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		gtx.Constraints.Max.Y = h
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(14), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Grup baru")
					l.Color, l.Font.Weight = u.t.Text, font.SemiBold
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { // nama grup
				return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return u.gcField(gtx, &u.gcNameEd, "Nama grup")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(16), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 12.5, itoa(selCount)+" anggota dipilih")
					l.Color = u.t.Text2
					return l.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(u.contactSendCache) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 14, "Tak ada kontak")
						l.Color = u.t.Text2
						return l.Layout(gtx)
					})
				}
				return material.List(u.th, &u.gcList).Layout(gtx, len(u.contactSendCache), func(gtx layout.Context, i int) layout.Dimensions {
					c := u.contactSendCache[i]
					return u.gcClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.gcContactRow(gtx, c, u.gcSel[c.JID])
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(14), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							b := material.Button(u.th, &u.gcCancel, "Batal")
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							b := material.Button(u.th, &u.gcCreate, "Buat")
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
					)
				})
			}),
		)
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

func (u *UI) gcField(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		e := material.Editor(u.th, ed, hint)
		e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(14.5)
		return e.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(8)
	paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// gcContactRow — baris kontak buat-grup: avatar + nama + tanda centang bila dipilih.
func (u *UI) gcContactRow(gtx layout.Context, c app.ContactRowDTO, sel bool) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, c.Name, c.JID, 38) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 15, c.Name)
				l.Color, l.MaxLines = u.t.Text, 1
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(22)
				sz := image.Pt(d, d)
				if sel {
					paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
					layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "check", 14, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
					})
				} else {
					paint.FillShape(gtx.Ops, u.t.Line, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					bw := gtx.Dp(2)
					in := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(d-bw, d-bw)}
					paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Ellipse{Min: in.Min, Max: in.Max}.Op(gtx.Ops))
				}
				return layout.Dimensions{Size: sz}
			}),
		)
	})
}

// contactSendLayer — modal pilih kontak utk dikirim (SendContact).
func (u *UI) contactSendLayer(gtx layout.Context) {
	if u.core != nil && u.contactSendCache == nil {
		u.contactSendCache = u.core.GetContacts()
		sort.Slice(u.contactSendCache, func(i, j int) bool {
			return strings.ToLower(u.contactSendCache[i].Name) < strings.ToLower(u.contactSendCache[j].Name)
		})
	}
	if len(u.contactSendClicks) < len(u.contactSendCache) {
		u.contactSendClicks = make([]widget.Clickable, len(u.contactSendCache))
	}
	for u.contactSendCancel.Clicked(gtx) {
		u.overlay = ""
		u.contactSendCache = nil
	}
	for i := range u.contactSendCache {
		if i >= len(u.contactSendClicks) {
			break
		}
		for u.contactSendClicks[i].Clicked(gtx) {
			c := u.contactSendCache[i]
			if u.core != nil && u.selected != "" {
				u.core.SendContact(u.selected, c.Name, c.Phone)
				u.messages = u.core.GetMessages(u.selected)
			}
			u.overlay = ""
			u.contactSendCache = nil
		}
	}
	u.contactSendList.Axis = layout.Vertical
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(380), gtx.Dp(460)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		gtx.Constraints.Max.Y = h
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 17, "Kirim kontak")
							l.Color, l.Font.Weight = u.t.Text, font.SemiBold
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.contactSendCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 20, u.t.Text2)
							})
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(u.contactSendCache) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 14, "Tak ada kontak")
						l.Color = u.t.Text2
						return l.Layout(gtx)
					})
				}
				return material.List(u.th, &u.contactSendList).Layout(gtx, len(u.contactSendCache), func(gtx layout.Context, i int) layout.Dimensions {
					c := u.contactSendCache[i]
					return u.contactSendClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, c.Name, c.JID, 40) }),
								layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(u.th, 15, c.Name)
											l.Color, l.MaxLines = u.t.Text, 1
											return l.Layout(gtx)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(u.th, 13, c.Phone)
											l.Color, l.MaxLines = u.t.Text2, 1
											return l.Layout(gtx)
										}),
									)
								}),
							)
						})
					})
				})
			}),
		)
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// locComposeLayer — modal susun lokasi: nama tempat + lat + lng → SendLocation.
func (u *UI) locComposeLayer(gtx layout.Context) {
	for u.locCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for u.locSend.Clicked(gtx) {
		name := strings.TrimSpace(u.locNameEd.Text())
		lat, _ := strconv.ParseFloat(strings.TrimSpace(u.locLatEd.Text()), 64)
		lng, _ := strconv.ParseFloat(strings.TrimSpace(u.locLngEd.Text()), 64)
		if u.core != nil && u.selected != "" && (name != "" || lat != 0 || lng != 0) {
			u.core.SendLocation(u.selected, lat, lng, name)
			u.locNameEd.SetText("")
			u.locLatEd.SetText("")
			u.locLngEd.SetText("")
			u.messages = u.core.GetMessages(u.selected)
		}
		u.overlay = ""
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(360), gtx.Dp(360)
		return composeCard(gtx, u.th, u.t, "Bagikan lokasi", []composeField{
			{&u.locNameEd, "Nama tempat (mis. Kantor)"},
			{&u.locLatEd, "Lintang (lat, mis. -6.2088)"},
			{&u.locLngEd, "Bujur (lng, mis. 106.8456)"},
		}, &u.locCancel, &u.locSend, "Kirim")
	})
}

// composeField + composeCard — kartu modal generik (judul + input + Batal/aksi).
type composeField struct {
	ed   *widget.Editor
	hint string
}

func composeCard(gtx layout.Context, th *material.Theme, t Theme, title string, fields []composeField, cancel, action *widget.Clickable, actionLabel string) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(th, ed, hint)
			e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(14.5)
			return e.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 17, title)
				l.Color, l.Font.Weight = t.Text, font.SemiBold
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		}
		for i := range fields {
			f := fields[i]
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, f.ed, f.hint) }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(th, cancel, "Batal")
					b.Background, b.Color, b.CornerRadius, b.TextSize = t.Bg2, t.Text, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(th, action, actionLabel)
					b.Background, b.Color, b.CornerRadius, b.TextSize = t.Accent, white, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pollComposeLayer — modal susun polling: pertanyaan + 4 opsi + Buat/Batal.
func (u *UI) pollComposeLayer(gtx layout.Context) {
	for u.pollCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for u.pollCreate.Clicked(gtx) {
		q := strings.TrimSpace(u.pollQEd.Text())
		var opts []string
		for i := range u.pollOptEds {
			if o := strings.TrimSpace(u.pollOptEds[i].Text()); o != "" {
				opts = append(opts, o)
			}
		}
		if q != "" && len(opts) >= 2 && u.core != nil && u.selected != "" {
			u.core.SendPoll(u.selected, q, opts, 1)
			u.pollQEd.SetText("")
			for i := range u.pollOptEds {
				u.pollOptEds[i].SetText("")
			}
			u.messages = u.core.GetMessages(u.selected)
		}
		u.overlay = ""
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(380), gtx.Dp(380)
		return u.pollComposeCard(gtx)
	})
}

func (u *UI) pollComposeCard(gtx layout.Context) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(u.th, ed, hint)
			e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(14.5)
			return e.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 17, "Buat polling")
				l.Color, l.Font.Weight = u.t.Text, font.SemiBold
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.pollQEd, "Pertanyaan") }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		}
		for i := range u.pollOptEds {
			ed := &u.pollOptEds[i]
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, ed, "Opsi") }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(u.th, &u.pollCancel, "Batal")
					b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(u.th, &u.pollCreate, "Buat")
					b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, white, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// fwdCtl membangun controller modal teruskan dari daftar chat nyata.
func (u *UI) fwdCtl() *FwdCtl {
	rows := make([]mvRow, len(u.chats))
	for i, c := range u.chats {
		rows[i] = mvRow{name: c.Name, sub: c.Preview}
	}
	if len(u.fwdClicks) < len(u.chats) {
		u.fwdClicks = make([]widget.Clickable, len(u.chats))
	}
	return &FwdCtl{Rows: rows, Clicks: u.fwdClicks, Cancel: &u.fwdCancel}
}

// handleReaction memproses klik emoji di pemilih reaksi: bila ada target pesan →
// core.React; bila tidak (dibuka dari tombol emoji composer) → sisipkan ke editor.
func (u *UI) handleReaction(gtx layout.Context) {
	for i := range u.rpClicks {
		for u.rpClicks[i].Clicked(gtx) {
			glyph := RpEmoji()[i]
			if u.reactMsgID != "" {
				if u.core != nil {
					u.core.React(u.selected, u.reactMsgID, u.reactSender, glyph, u.reactFromMe)
				}
				u.reactMsgID = ""
			} else {
				u.editor.Insert(glyph) // sisip emoji ke pesan yg sedang diketik
			}
			u.overlay = ""
		}
	}
}

// replyDisplayName — nama yg ditampilkan di banner balas.
func (u *UI) replyDisplayName(m app.MessageDTO) string {
	if m.Dir == "out" {
		return "Anda"
	}
	if m.Sender != "" {
		return m.Sender
	}
	return u.selName
}

// ctxMenuView — menu aksi pesan (.menu): kartu bg + baris glyph+label klik.
func (u *UI) ctxMenuView(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(ctxMenu))
	for i := range ctxMenu {
		i := i
		it := ctxMenu[i]
		for u.ctxItems[i].Clicked(gtx) {
			u.doCtxAction(it.label) // jalankan aksi engine bila ada
			u.overlay = it.to       // pindah ke popup tujuan ("" = tutup)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.ctxItems[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					dcol := u.t.Text
					if it.label == "Hapus" {
						dcol = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, it.icon, 18, dcol) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 14.5, it.label)
							lbl.Color = dcol
							return lbl.Layout(gtx)
						}),
					)
				})
			})
		}))
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	call := macro.Stop()
	rr := gtx.Dp(10)
	paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// chatTag — tag pointer per baris chat (deteksi klik-kanan).
type chatTag int

// chatCtxAction = item menu aksi baris chat (ikon, label, aksi, danger).
type chatCtxAction struct {
	icon, label, action string
	danger              bool
}

// chatCtxActions menghasilkan item menu menurut status chat (pin/mute toggle).
func chatCtxActions(c app.ChatDTO) []chatCtxAction {
	pin, mute := "Sematkan", "Bisukan"
	if c.Pinned {
		pin = "Lepas sematan"
	}
	if c.Muted {
		mute = "Bunyikan"
	}
	return []chatCtxAction{
		{"pin", pin, "pin", false},
		{"mute", mute, "mute", false},
		{"archive", "Arsipkan", "archive", false},
		{"message", "Tandai belum dibaca", "unread", false},
		{"trash", "Hapus chat", "delete", true},
	}
}

// doChatAction menjalankan aksi baris chat terhadap engine + refresh bila perlu.
func (u *UI) doChatAction(action string, c app.ChatDTO) {
	if u.core == nil {
		return
	}
	switch action {
	case "pin":
		u.core.Pin(c.ID, !c.Pinned)
	case "mute":
		u.core.Mute(c.ID, !c.Muted)
	case "archive":
		u.core.Archive(c.ID, true)
	case "unread":
		u.core.MarkUnread(c.ID, true)
	case "delete":
		u.core.DeleteChat(c.ID)
	}
	u.chats = u.core.GetChats()
}

// chatCtxView — menu aksi baris chat (klik-kanan): kartu + baris glyph+label.
func (u *UI) chatCtxView(gtx layout.Context) layout.Dimensions {
	if u.chatCtxChat.ID == "" {
		u.overlay = ""
		return layout.Dimensions{}
	}
	c := u.chatCtxChat // snapshot saat menu dibuka
	items := chatCtxActions(c)
	children := make([]layout.FlexChild, 0, len(items))
	for i := range items {
		i := i
		it := items[i]
		for u.chatCtxItems[i].Clicked(gtx) {
			u.doChatAction(it.action, c)
			u.overlay = ""
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.chatCtxItems[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					dcol := u.t.Text
					if it.danger {
						dcol = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, it.icon, 18, dcol) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 14.5, it.label)
							lbl.Color = dcol
							return lbl.Layout(gtx)
						}),
					)
				})
			})
		}))
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	call := macro.Stop()
	rr := gtx.Dp(10)
	paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// ---- rail (nav kiri, tombol klik → ganti view) ----
func (u *UI) rail(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(56)
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, u.t.RailBg, clip.Rect{Max: sz}.Op())
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	children := []layout.FlexChild{layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout)}
	for i := range railNav {
		i := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.railBtn(gtx, i)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
	}
	layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx, children...)
	return layout.Dimensions{Size: sz}
}

func (u *UI) railBtn(gtx layout.Context, i int) layout.Dimensions {
	nav := railNav[i]
	for u.railClicks[i].Clicked(gtx) {
		u.view = nav.view
	}
	return u.railClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(44)
		sz := image.Pt(d, d)
		active := u.view == nav.view
		rad := d / 2
		bg := color.NRGBA{}
		if active {
			bg = color.NRGBA{R: 0, G: 168, B: 132, A: 38}
			rad = gtx.Dp(14)
		} else if u.railClicks[i].Hovered() {
			bg = u.t.Hover
		}
		if bg.A > 0 {
			paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: rad, NE: rad, SE: rad, SW: rad}.Op(gtx.Ops))
		}
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		col := u.t.RailIco
		if active {
			col = u.t.Accent
		}
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, nav.icon, 24, col)
		})
		return layout.Dimensions{Size: sz}
	})
}

// ---- sidebar (dispatch per view: settings/calls pane, else daftar chat) ----
func (u *UI) sidebar(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(380)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	switch u.view {
	case "settings":
		u.handleSettings(gtx)
		kd := true
		if u.core != nil {
			kd = u.core.GetKeepDeleted()
		}
		ctl := &SettingsCtl{
			Dark: u.dark, KeepDeleted: kd, Clicks: u.setClicks[:],
			Sub: u.setSub, Back: &u.setBack, ProfileClick: &u.setProfileClick,
		}
		if u.setSub != "" && u.core != nil { // data sub-pane
			switch u.setSub {
			case "profile":
				p := u.core.GetProfile()
				ctl.ProfName, ctl.ProfAbout, ctl.ProfPhone = p.Name, p.About, p.Phone
				if !u.profLoaded { // isi editor sekali (agar bisa diketik)
					u.profNameEd.SetText(p.Name)
					u.profAboutEd.SetText(p.About)
					u.profLoaded = true
				}
				ctl.ProfNameEd, ctl.ProfAboutEd, ctl.ProfSave = &u.profNameEd, &u.profAboutEd, &u.profSave
			case "storage":
				s := u.core.GetStorageUsage()
				ctl.StoreDB, ctl.StoreMedia, ctl.StoreMsgs = s.DBBytes, s.MediaBytes, s.MsgCount
			case "privacy":
				pv := u.core.GetPrivacy()
				ctl.Privacy = pv
				ctl.PrivacyClicks = u.privacyClicks[:]
				for i := range privacyOrder { // ketuk baris → siklus all→contacts→none
					if u.privacyClicks[i].Clicked(gtx) {
						k := privacyOrder[i].key
						u.core.SetPrivacy(k, nextPrivacy(pv[k]))
					}
				}
			}
		}
		return SettingsView(gtx, u.th, u.t, ctl)
	case "calls":
		return SidePanesView(gtx, u.th, u.t, u.callRows())
	case "contacts":
		groups := u.contactGroups()
		u.handleContactsPane(gtx)
		for u.gcNewBtn.Clicked(gtx) { // "Grup baru" → modal buat grup
			u.overlay = "groupcreate"
		}
		return ContactsPaneView(gtx, u.th, u.t, groups, u.contactPaneClicks, &u.gcNewBtn)
	case "status":
		items := u.statusRows()
		u.handleStatus(gtx)
		return StatusPaneView(gtx, u.th, u.t, items, u.statusClicks)
	case "channels":
		return ChannelsPaneView(gtx, u.th, u.t, u.channelRows())
	}
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Rect{Max: sz}.Op())

	u.handleChatFilter(gtx)
	u.computeShown()
	u.handleNewChat(gtx)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.header(gtx, w, "Chat", u.t.Text, 23, font.Bold)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.searchBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.filterChips(gtx)
		}),
		// query berupa nomor telepon → tawarkan "mulai chat baru".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ph := phoneQuery(u.searchEd.Text()); ph != "" {
				return u.newChatRow(gtx, ph)
			}
			return layout.Dimensions{}
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(u.th, &u.chatList).Layout(gtx, len(u.shown), func(gtx layout.Context, i int) layout.Dimensions {
				return u.chatRow(gtx, u.shown[i])
			})
		}),
	)
}

// phoneQuery — kembalikan digit nomor bila teks pencarian "mirip nomor" (≥8 digit,
// hanya digit/+/spasi/strip), atau "" bila bukan. Utk tawaran mulai-chat-baru.
func phoneQuery(s string) string {
	s = strings.TrimSpace(s)
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits = append(digits, r)
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return "" // ada huruf → bukan nomor
		}
	}
	if len(digits) < 8 {
		return ""
	}
	return string(digits)
}

// handleNewChat — klik baris "mulai chat baru" → buka chat dgn JID nomor itu.
func (u *UI) handleNewChat(gtx layout.Context) {
	for u.newChatClick.Clicked(gtx) {
		ph := phoneQuery(u.searchEd.Text())
		if ph == "" {
			continue
		}
		jid := ph + "@s.whatsapp.net"
		u.selected, u.selName, u.selGroup = jid, "+"+ph, false
		u.searchEd.SetText("")
		if u.core != nil {
			u.core.OpenChat(jid)
			u.messages = u.core.GetMessages(jid)
		}
		u.msgList.ScrollTo(len(u.messages))
	}
}

// newChatRow — baris tawaran "Mulai chat baru" dgn nomor (ikon newchat accent).
func (u *UI) newChatRow(gtx layout.Context, ph string) layout.Dimensions {
	return u.newChatClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(20), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := gtx.Dp(40)
					sz := image.Pt(d, d)
					paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
					layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "message", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
					})
					return layout.Dimensions{Size: sz}
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 15.5, "Mulai chat baru")
							l.Color, l.Font.Weight = u.t.Text, font.Medium
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 13.5, "+"+ph)
							l.Color = u.t.Accent
							return l.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

// chatFilterLabels — chip filter daftar chat (paritas Filters.svelte).
var chatFilterLabels = []string{"Semua", "Belum dibaca", "Favorit", "Grup"}

// handleChatFilter memproses klik chip filter.
func (u *UI) handleChatFilter(gtx layout.Context) {
	for i := range u.filterClicks {
		for u.filterClicks[i].Clicked(gtx) {
			u.filterSel = i
		}
	}
}

// computeShown menyaring u.chats menurut filter aktif + teks pencarian → u.shown.
func (u *UI) computeShown() {
	q := strings.ToLower(strings.TrimSpace(u.searchEd.Text()))
	u.shown = u.shown[:0]
	for i, c := range u.chats {
		switch u.filterSel {
		case 1: // Belum dibaca
			if !c.Unread && c.Badge == 0 {
				continue
			}
		case 2: // Favorit (disematkan)
			if !c.Pinned {
				continue
			}
		case 3: // Grup
			if !c.Group {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Preview), q) {
			continue
		}
		u.shown = append(u.shown, i)
	}
}

// searchBar — input pencarian membulat (ikon + editor) di var(--search-bg).
func (u *UI) searchBar(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(8), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "search", 18, u.t.Text2)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(u.th, &u.searchEd, "Cari atau mulai chat baru")
					e.TextSize = unit.Sp(14)
					e.Color = u.t.Text
					e.HintColor = u.t.Text2
					return e.Layout(gtx)
				}),
			)
		})
		call := macro.Stop()
		r := gtx.Dp(18)
		w := gtx.Constraints.Max.X
		bg := clip.RRect{Rect: image.Rectangle{Max: image.Pt(w, dims.Size.Y)}, NW: r, NE: r, SE: r, SW: r}
		paint.FillShape(gtx.Ops, u.t.SearchBg, bg.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return layout.Dimensions{Size: image.Pt(w, dims.Size.Y)}
	})
}

// filterChips — baris chip filter (aktif = bg accent lembut + teks accent).
func (u *UI) filterChips(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12), Top: unit.Dp(2), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(chatFilterLabels)*2)
		for i := range chatFilterLabels {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
			}
			idx := i
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.filterChip(gtx, idx)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

func (u *UI) filterChip(gtx layout.Context, i int) layout.Dimensions {
	active := u.filterSel == i
	txtCol := u.t.Text2
	chipBg := u.t.SearchBg
	if active {
		txtCol = color.NRGBA{R: 0x00, G: 0xa8, B: 0x84, A: 0xff} // accent
		chipBg = color.NRGBA{R: 0x00, G: 0xa8, B: 0x84, A: 0x2e} // accent lembut
	}
	return u.filterClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 13, chatFilterLabels[i])
			lbl.Color = txtCol
			return lbl.Layout(gtx)
		})
		call := macro.Stop()
		r := dims.Size.Y / 2
		paint.FillShape(gtx.Ops, chipBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

func (u *UI) header(gtx layout.Context, w int, title string, col color.NRGBA, sp unit.Sp, wt font.Weight) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	// divider bawah
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Inset{Left: unit.Dp(18), Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(u.th, sp, title)
		lbl.Color = col
		lbl.Font.Weight = wt
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// ---- baris chat (.chat-row) ----
func (u *UI) chatRow(gtx layout.Context, i int) layout.Dimensions {
	c := u.chats[i]
	active := c.ID == u.selected
	return u.clicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		for u.clicks[i].Clicked(gtx) {
			u.selected = c.ID
			u.selName = c.Name
			u.selGroup = c.Group
			if u.core != nil {
				u.core.OpenChat(c.ID)
				u.messages = u.core.GetMessages(c.ID)
				u.prefetchHistory(c.ID) // history tipis → backfill lama terurut
			}
			u.msgList.ScrollTo(len(u.messages)) // buka chat → ke pesan terbaru (bawah)
		}
		// bg hover/active
		dims := layout.UniformInset(0).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// .chat-list pad 4/8 + .chat-row pad 10/12 → vert 10, horiz 8+12=20.
			return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.avatar(gtx, c.Name, c.ID, 49)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return u.rowLine(gtx, c.Name, c.Time, 16.5, u.t.Text, u.t.Text2)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return u.previewLine(gtx, c)
								}),
							)
						}),
					)
				})
			})
		})
		_ = active
		// klik-kanan (secondary) di baris → menu aksi chat.
		tag := chatTag(i)
		for {
			ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press})
			if !ok {
				break
			}
			if pe, ok := ev.(pointer.Event); ok && pe.Buttons.Contain(pointer.ButtonSecondary) {
				u.chatCtxChat = c // snapshot chat
				u.overlay = "chatctx"
			}
		}
		area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
		event.Op(gtx.Ops, tag)
		area.Pop()
		return dims
	})
}

func (u *UI) rowLine(gtx layout.Context, name, t string, sp unit.Sp, nameCol, timeCol color.NRGBA) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, sp, name)
			lbl.Color = nameCol
			lbl.MaxLines = 1
			lbl.Font.Weight = font.Medium
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 12, t)
			lbl.Color = timeCol
			return lbl.Layout(gtx)
		}),
	)
}

func (u *UI) previewLine(gtx layout.Context, c app.ChatDTO) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 14, c.Preview)
			lbl.Color = u.t.Text2
			lbl.MaxLines = 1
			return lbl.Layout(gtx)
		}),
		// indikator: bisu (mute) + sematkan (pin) + badge belum-dibaca.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !c.Muted {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "mute", 16, u.t.Text2) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !c.Pinned {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "pin", 16, u.t.Text2) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.Badge <= 0 {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return u.badge(gtx, c.Badge) })
		}),
	)
}

func (u *UI) badge(gtx layout.Context, n int) layout.Dimensions {
	r := gtx.Dp(10)
	d := r * 2
	paint.FillShape(gtx.Ops, u.t.Accent, clip.RRect{Rect: image.Rectangle{Max: image.Pt(d, d)}, SE: r, SW: r, NW: r, NE: r}.Op(gtx.Ops))
	gtx.Constraints.Min = image.Pt(d, d)
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(u.th, 11, itoa(n))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: image.Pt(d, d)}
}

// ---- avatar (lingkaran warna + inisial) ----
// ensureAvatar memuat foto profil (engine AvatarBytes) sekali per jid di goroutine
// latar → decode → cache di u.photos[name]. Tak memblok thread UI; sekali gagal
// tetap ditandai agar tak refetch terus.
func (u *UI) ensureAvatar(name, jid string) {
	if u.core == nil || jid == "" {
		return
	}
	u.photoMu.Lock()
	if u.photoTried[jid] {
		u.photoMu.Unlock()
		return
	}
	u.photoTried[jid] = true
	u.photoMu.Unlock()
	go func() {
		b := u.core.AvatarBytes(jid)
		img := decodeImage(b)
		if img == nil {
			return
		}
		op := paint.NewImageOp(img)
		u.photoMu.Lock()
		u.photos[name] = op
		u.photoMu.Unlock()
	}()
}

// reactionPills — chip reaksi di bawah bubble (emoji + jumlah bila >1). Pil Bg2
// membulat; milik-sendiri di-beri batas accent. Kosong bila tak ada reaksi.
func (u *UI) reactionPills(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	if len(m.Reactions) == 0 {
		return layout.Dimensions{}
	}
	return layout.Inset{Top: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(m.Reactions)*2)
		for i, rx := range m.Reactions {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout))
			}
			rx := rx
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				txt := rx.Emoji
				if rx.Count > 1 {
					txt = rx.Emoji + " " + itoa(rx.Count)
				}
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 13, txt)
					lbl.Color = u.t.Text
					return lbl.Layout(gtx)
				})
				call := macro.Stop()
				r := dims.Size.Y / 2
				border := u.t.Bg2
				if rx.Mine {
					border = u.t.Accent
				}
				paint.FillShape(gtx.Ops, border, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				bw := gtx.Dp(1)
				inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
				paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

// locationBubble — kartu lokasi: kotak peta (placeholder Bg2 + ikon locpin) +
// baris alamat (locpin + m.Text). Tap → buka peta (follow-up).
func (u *UI) locationBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	addr := m.Text
	if addr == "" {
		addr = "Lokasi"
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			w := gtx.Dp(220)
			h := gtx.Dp(120)
			box := image.Pt(w, h)
			r := gtx.Dp(10)
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "locpin", 30, u.t.Accent) })
			return layout.Dimensions{Size: box}
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "locpin", 15, u.t.Text2) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14, addr)
					lbl.Color = u.t.Text
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
	)
}

// contactBubble — kartu kontak: avatar + nama (m.Text) + tautan "Simpan" accent.
func (u *UI) contactBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	name := m.Text
	if name == "" {
		name = "Kontak"
	}
	sub := m.Thumb // nomor telepon bila ada
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, name, "", 40) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14.5, name)
					lbl.Color = u.t.Text
					lbl.Font.Weight = font.Medium
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if sub == "" {
						return layout.Dimensions{}
					}
					lbl := material.Label(u.th, 13, sub)
					lbl.Color = u.t.Text2
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 13, "Simpan")
			lbl.Color = u.t.Accent
			return lbl.Layout(gtx)
		}),
	)
}

// docBubble — kartu dokumen: ikon docfile (kotak accent) + nama berkas + sub
// (PDF · ukuran · halaman). Tap → OnPlayVideo? tidak; dokumen dibuka via engine
// (follow-up). m.Text = nama berkas; DocMime/DocSize/DocPages utk sub.
func (u *UI) docBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	name := m.Text
	if name == "" {
		name = "Dokumen"
	}
	var parts []string
	if ext := docExt(m.DocMime); ext != "" {
		parts = append(parts, ext)
	}
	if m.DocSize > 0 {
		parts = append(parts, fmtBytes(m.DocSize))
	}
	if m.DocPages > 0 {
		parts = append(parts, itoa(m.DocPages)+" hal")
	}
	sub := strings.Join(parts, " · ")
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			d := gtx.Dp(40)
			sz := image.Pt(d, d)
			r := gtx.Dp(8)
			paint.FillShape(gtx.Ops, withAlpha(u.t.Accent, 0x33), clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "docfile", 22, u.t.Accent) })
			return layout.Dimensions{Size: sz}
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14.5, name)
					lbl.Color = u.t.Text
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if sub == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 12.5, sub)
						lbl.Color = u.t.Text2
						return lbl.Layout(gtx)
					})
				}),
			)
		}),
	)
}

// voiceBubble — pesan suara: tombol play + waveform (batang) + durasi. Tap pada
// bubble memutar (OnPlayVoice di bubble()).
func (u *UI) voiceBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "play", 26, u.t.Accent) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.waveform(gtx) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dur := m.Text
			if dur == "" {
				dur = "0:08"
			}
			lbl := material.Label(u.th, 11, dur)
			lbl.Color = u.t.Text2
			return lbl.Layout(gtx)
		}),
	)
}

// waveform — batang-batang tinggi bervariasi (visual statis pesan suara).
func (u *UI) waveform(gtx layout.Context) layout.Dimensions {
	heights := []int{6, 11, 8, 14, 9, 16, 7, 12, 10, 15, 6, 13, 8, 11, 7}
	bw := gtx.Dp(2)
	gap := gtx.Dp(2)
	maxH := gtx.Dp(18)
	w := len(heights) * (bw + gap)
	for i, h := range heights {
		hp := gtx.Dp(unit.Dp(h))
		x := i * (bw + gap)
		y := (maxH - hp) / 2
		paint.FillShape(gtx.Ops, u.t.Text2, clip.RRect{Rect: image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+hp)}, NW: bw / 2, NE: bw / 2, SE: bw / 2, SW: bw / 2}.Op(gtx.Ops))
	}
	return layout.Dimensions{Size: image.Pt(w, maxH)}
}

// fmtBytes — ukuran berkas ringkas.
func fmtBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return itoa(int(n>>20)) + " MB"
	case n >= 1<<10:
		return itoa(int(n>>10)) + " KB"
	default:
		return itoa(int(n)) + " B"
	}
}

// docExt — label jenis dari mime ("application/pdf"→"PDF").
func docExt(mime string) string {
	// urut: spreadsheet & word DULU sebelum "document" generik — sebab mime OOXML
	// (xlsx/docx) sama-sama mengandung "officedocument".
	switch {
	case mime == "":
		return ""
	case strings.Contains(mime, "pdf"):
		return "PDF"
	case strings.Contains(mime, "sheet") || strings.Contains(mime, "excel"):
		return "XLS"
	case strings.Contains(mime, "word"):
		return "DOC"
	case strings.Contains(mime, "zip"):
		return "ZIP"
	}
	return "FILE"
}

// withAlpha — warna dgn alpha berbeda.
func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// mentionText — render teks dgn @mention berwarna accent (richtext, wrap benar).
func (u *UI) mentionText(gtx layout.Context, text string, mentions []app.MentionDTO) layout.Dimensions {
	base := richtext.SpanStyle{Size: unit.Sp(15), Color: u.t.Text}
	acc := base
	acc.Color = u.t.Accent
	acc.Font.Weight = font.Medium
	spans := mentionSpans(text, mentions, base, acc)
	return richtext.Text(&u.mentionState, u.th.Shaper, spans...).Layout(gtx)
}

// mentionSpans — pisah `text` jadi span normal vs span accent pada token "@Name"
// (dari mentions). Pure → bisa diuji. Token paling-awal di tiap posisi yg dipilih.
func mentionSpans(text string, mentions []app.MentionDTO, base, acc richtext.SpanStyle) []richtext.SpanStyle {
	toks := make([]string, 0, len(mentions))
	for _, mn := range mentions {
		if mn.Name != "" {
			toks = append(toks, "@"+mn.Name)
		}
	}
	var spans []richtext.SpanStyle
	for i := 0; i < len(text); {
		bestPos, bestTok := -1, ""
		for _, tok := range toks {
			if p := strings.Index(text[i:], tok); p >= 0 && (bestPos < 0 || p < bestPos) {
				bestPos, bestTok = p, tok
			}
		}
		if bestPos < 0 {
			s := base
			s.Content = text[i:]
			spans = append(spans, s)
			break
		}
		if bestPos > 0 {
			s := base
			s.Content = text[i : i+bestPos]
			spans = append(spans, s)
		}
		s := acc
		s.Content = bestTok
		spans = append(spans, s)
		i += bestPos + len(bestTok)
	}
	return spans
}

// pollVoteEntry — hasil suara ter-cache + waktunya (TTL 2s).
type pollVoteEntry struct {
	v  app.PollVotesDTO
	at time.Time
}

// cachedPollVotes — GetPollVotes dgn TTL 2s per poll → tak query DB tiap frame
// (mis. saat scroll). Suara baru muncul ≤2s, cukup cepat utuk UI.
func (u *UI) cachedPollVotes(msgID string) app.PollVotesDTO {
	if e, ok := u.pollVoteCache[msgID]; ok && time.Since(e.at) < 2*time.Second {
		return e.v
	}
	v := u.core.GetPollVotes(msgID)
	u.pollVoteCache[msgID] = pollVoteEntry{v: v, at: time.Now()}
	return v
}

// pollBubble — kartu polling: ikon+pertanyaan (m.Text) + opsi bordered (m.Thumb =
// JSON []string). Tampilan (voting = follow-up). Paritas .poll-card.
func (u *UI) pollBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	var opts []string
	if m.Thumb != "" {
		_ = json.Unmarshal([]byte(m.Thumb), &opts)
	}
	// hasil suara (counts/total) + clickable per opsi.
	var counts map[string]int
	total := 0
	if u.core != nil {
		v := u.cachedPollVotes(m.ID)
		counts, total = v.Counts, v.Total
	}
	clks := u.pollClicks[m.ID]
	if len(clks) < len(opts) {
		clks = make([]widget.Clickable, len(opts))
		u.pollClicks[m.ID] = clks
	}
	pollSender := m.SenderID
	if pollSender == "" {
		pollSender = m.Sender
	}
	children := make([]layout.FlexChild, 0, len(opts)*2+1)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "pollq", 16, u.t.Text2) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 14.5, m.Text)
				lbl.Color = u.t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
		)
	}))
	for i := range opts {
		o := opts[i]
		clk := &u.pollClicks[m.ID][i]
		for clk.Clicked(gtx) { // tap opsi → kirim suara
			if u.core != nil {
				u.core.VotePoll(u.selected, pollSender, m.ID, []string{o})
				delete(u.pollVoteCache, m.ID) // invalidasi → hitung baru tampil segera
			}
		}
		cnt := counts[o]
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return u.pollOption(gtx, o, cnt, total)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// pollOption — opsi polling: kotak bordered + radio + label + jumlah suara, dgn bar
// proporsi (cnt/total) berlatar accent lembut.
func (u *UI) pollOption(gtx layout.Context, label string, cnt, total int) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(18)
				sz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, u.t.Line, clip.Ellipse{Max: sz}.Op(gtx.Ops))
				bw := gtx.Dp(2)
				in := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(d-bw, d-bw)}
				paint.FillShape(gtx.Ops, u.t.InBg, clip.Ellipse{Min: in.Min, Max: in.Max}.Op(gtx.Ops))
				return layout.Dimensions{Size: sz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 14, label)
				lbl.Color = u.t.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if total <= 0 {
					return layout.Dimensions{}
				}
				lbl := material.Label(u.th, 13, itoa(cnt))
				lbl.Color = u.t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(8)
	paint.FillShape(gtx.Ops, u.t.Line, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	bw := gtx.Dp(1)
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
	paint.FillShape(gtx.Ops, u.t.InBg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	if total > 0 && cnt > 0 { // bar proporsi accent lembut
		bw2 := (inner.Dx() * cnt) / total
		bar := image.Rectangle{Min: inner.Min, Max: image.Pt(inner.Min.X+bw2, inner.Max.Y)}
		paint.FillShape(gtx.Ops, withAlpha(u.t.Accent, 0x22), clip.RRect{Rect: bar, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	}
	call.Add(gtx.Ops)
	return dims
}

// quoteBlock — kutipan pesan yg dibalas, di dalam bubble (garis accent kiri + nama
// + teks), latar agak gelap. margin-bottom kecil sebelum isi.
func (u *UI) quoteBlock(gtx layout.Context, m app.MessageDTO, out bool) layout.Dimensions {
	name := m.QuoteName
	if name == "" {
		name = "Pesan"
	}
	return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 13, name)
					lbl.Color = u.t.Accent
					lbl.Font.Weight = font.Medium
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 13, m.QuoteText)
					lbl.Color = u.t.Text2
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		})
		call := macro.Stop()
		r := gtx.Dp(5)
		// latar kutipan: sedikit kontras dari bubble.
		qbg := u.t.InBg
		if out {
			qbg = color.NRGBA{R: 0, G: 0, B: 0, A: 40}
		} else {
			qbg = color.NRGBA{R: 0, G: 0, B: 0, A: 30}
		}
		paint.FillShape(gtx.Ops, qbg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

// ensureMedia memuat byte media bubble (engine MediaBytes) sekali per msgID di
// goroutine → decode → cache u.media[id]. Tak memblok UI.
func (u *UI) ensureMedia(chat, id string) {
	if u.core == nil || id == "" {
		return
	}
	u.mediaMu.Lock()
	if u.mediaTried[id] {
		u.mediaMu.Unlock()
		return
	}
	u.mediaTried[id] = true
	u.mediaMu.Unlock()
	go func() {
		b := u.core.MediaBytes(chat, id)
		img := decodeImage(b)
		if img == nil {
			return
		}
		op := paint.NewImageOp(img)
		u.mediaMu.Lock()
		u.media[id] = op
		u.mediaMu.Unlock()
	}()
}

// mediaThumb — thumbnail media bubble (image/video/gif): kotak membulat 220px (rasio
// asli bila termuat, else 4:3 placeholder Bg2 + ikon), play-overlay utk video/gif,
// lalu caption m.Text bila ada. Tap di-tangani bubble (OnPlayVideo).
func (u *UI) mediaThumb(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	u.ensureMedia(u.selected, m.ID)
	u.mediaMu.Lock()
	op, ok := u.media[m.ID]
	u.mediaMu.Unlock()

	w := gtx.Dp(220)
	h := w * 3 / 4
	if ok {
		s := op.Size()
		if s.X > 0 && s.Y > 0 {
			h = w * s.Y / s.X
			if max := gtx.Dp(300); h > max {
				h = max
			}
		}
	}
	box := image.Pt(w, h)
	r := gtx.Dp(10)

	thumb := func(gtx layout.Context) layout.Dimensions {
		if ok {
			cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
			drawImageFill(gtx.Ops, op, w) // cover lebar; tinggi mengikuti
			_ = h
			cl.Pop()
		} else {
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			ic := "wallpaperico"
			if m.Type != "image" {
				ic = "play"
			}
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, ic, 36, u.t.Text2)
			})
		}
		// play-overlay (video/gif): lingkaran gelap + segitiga putih di tengah.
		if m.Type == "video" || m.Type == "gif" {
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(46)
				sz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, color.NRGBA{A: 140}, clip.Ellipse{Max: sz}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "play", 22, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
				})
				return layout.Dimensions{Size: sz}
			})
		}
		return layout.Dimensions{Size: box}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(thumb),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.Text == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 15, m.Text)
				lbl.Color = u.t.Text
				return lbl.Layout(gtx)
			})
		}),
	)
}

// avatar — foto profil bulat (jid → AvatarBytes, di-cache); fallback inisial warna.
func (u *UI) avatar(gtx layout.Context, name, jid string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	u.ensureAvatar(name, jid)
	// Foto in-memory (byte engine → ImageOp) di-mask bulat; else inisial.
	u.photoMu.Lock()
	ph, ok := u.photos[name]
	u.photoMu.Unlock()
	if ok {
		cl := clip.Ellipse{Max: sz}.Push(gtx.Ops)
		drawImageFill(gtx.Ops, ph, d)
		cl.Pop()
		return layout.Dimensions{Size: sz}
	}
	col := avatarColor(name)
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(u.th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// contactGroups membangun pane Kontak dari kontak nyata (core.GetContacts),
// dikelompokkan per huruf awal nama (urut). nil = mode demo.
func (u *UI) contactGroups() []cpGroup {
	if u.core == nil {
		return nil
	}
	if u.cgCache != nil && time.Since(u.cgAt) < time.Second {
		return u.cgCache // TTL: pertahankan contactFlat/clicks dari build terakhir
	}
	cs := u.core.GetContacts()
	sort.Slice(cs, func(i, j int) bool {
		return strings.ToLower(cs[i].Name) < strings.ToLower(cs[j].Name)
	})
	u.contactFlat = u.contactFlat[:0]
	var groups []cpGroup
	var cur *cpGroup
	for _, c := range cs {
		if c.Name == "" {
			continue
		}
		letter := strings.ToUpper(initial(c.Name))
		if cur == nil || cur.letter != letter {
			groups = append(groups, cpGroup{letter: letter})
			cur = &groups[len(groups)-1]
		}
		idx := len(u.contactFlat)
		u.contactFlat = append(u.contactFlat, c)
		cur.items = append(cur.items, cpContact{name: c.Name, about: c.Phone, idx: idx})
	}
	if len(u.contactPaneClicks) < len(u.contactFlat) {
		u.contactPaneClicks = make([]widget.Clickable, len(u.contactFlat))
	}
	u.cgCache, u.cgAt = groups, time.Now()
	return groups
}

// handleContactsPane — ketuk kontak → buka/mulai chat (pindah ke pane chats).
func (u *UI) handleContactsPane(gtx layout.Context) {
	for i := range u.contactFlat {
		if i >= len(u.contactPaneClicks) {
			break
		}
		for u.contactPaneClicks[i].Clicked(gtx) {
			c := u.contactFlat[i]
			u.selected, u.selName, u.selGroup = c.JID, c.Name, false
			u.view = "chats"
			if u.core != nil {
				u.core.OpenChat(c.JID)
				u.messages = u.core.GetMessages(c.JID)
				u.prefetchHistory(c.JID) // history tipis → backfill lama terurut
			}
			u.msgList.ScrollTo(len(u.messages))
		}
	}
}

// channelRows membangun pane Saluran dari saluran nyata (core.GetChannels). nil = demo.
func (u *UI) channelRows() []chnChannel {
	if u.core == nil {
		return nil
	}
	if u.chCache != nil && time.Since(u.chAt) < time.Second {
		return u.chCache
	}
	cs := u.core.GetChannels()
	out := make([]chnChannel, 0, len(cs))
	for _, c := range cs {
		out = append(out, chnChannel{name: c.Name, subs: fmtSubs(c.Subscribers)})
	}
	u.chCache, u.chAt = out, time.Now()
	return out
}

// fmtSubs — "12 pengikut" / "12,3 rb pengikut" / "1,2 jt pengikut".
func fmtSubs(n int) string {
	switch {
	case n >= 1000000:
		return itoa(n/1000000) + " jt pengikut"
	case n >= 1000:
		return itoa(n/1000) + " rb pengikut"
	default:
		return itoa(n) + " pengikut"
	}
}

// statusRows membangun baris pane Status (TERKINI) dari grup status nyata
// (core.GetStatuses), mengecualikan status sendiri (itu baris "My status"). nil =
// mode demo.
func (u *UI) statusRows() []stpItem {
	if u.core == nil {
		return nil
	}
	if u.srCache != nil && time.Since(u.srAt) < time.Second {
		return u.srCache
	}
	gs := u.core.GetStatuses()
	u.statusGroupsCache = u.statusGroupsCache[:0]
	out := make([]stpItem, 0, len(gs))
	for _, g := range gs {
		if g.Mine {
			continue // status sendiri tampil di baris My-status, bukan daftar
		}
		u.statusGroupsCache = append(u.statusGroupsCache, g)
		out = append(out, stpItem{name: g.Name, time: g.Time, seen: false})
	}
	if len(u.statusClicks) < len(out) {
		u.statusClicks = make([]widget.Clickable, len(out))
	}
	u.srCache, u.srAt = out, time.Now()
	return out
}

// statusViewLayer — penampil status layar penuh: bar atas (nama+waktu+tutup) +
// item pertama (teks besar di tengah / gambar dari thumb data-URI).
func (u *UI) statusViewLayer(gtx layout.Context) {
	if u.statusViewIdx < 0 || u.statusViewIdx >= len(u.statusGroupsCache) {
		u.overlay = ""
		return
	}
	g := u.statusGroupsCache[u.statusViewIdx]
	for u.statusClose.Clicked(gtx) {
		u.overlay = ""
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 0xee}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	var item app.StatusItemDTO
	if len(g.Items) > 0 {
		item = g.Items[len(g.Items)-1] // terbaru
	}
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// bar atas: avatar + nama + waktu + tutup
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, g.Name, g.Jid, 38) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 15, g.Name)
								l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
								l.Font.Weight = font.Medium
								return l.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 12, item.Time)
								l.Color = color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
								return l.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.statusClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 22, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
						})
					}),
				)
			})
		}),
		// isi: gambar (thumb data-URI) atau teks besar.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if img := decodeImage(decodeDataURI(item.Thumb)); img != nil {
					op := paint.NewImageOp(img)
					sz := op.Size()
					maxW, maxH := gtx.Dp(420), gtx.Constraints.Max.Y-gtx.Dp(80)
					w := maxW
					h := w * sz.Y / sz.X
					if h > maxH {
						h = maxH
						w = h * sz.X / sz.Y
					}
					box := image.Pt(w, h)
					cl := clip.Rect{Max: box}.Push(gtx.Ops)
					drawImageFill(gtx.Ops, op, w)
					cl.Pop()
					return layout.Dimensions{Size: box}
				}
				txt := item.Text
				if txt == "" {
					txt = "Status"
				}
				return layout.Inset{Left: unit.Dp(40), Right: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 26, txt)
					l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					l.Alignment = text.Middle
					return l.Layout(gtx)
				})
			})
		}),
	)
}

// decodeDataURI — "data:<mime>;base64,<...>" → byte. "" bila bukan data-URI base64.
func decodeDataURI(s string) []byte {
	i := strings.Index(s, ";base64,")
	if i < 0 || !strings.HasPrefix(s, "data:") {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(s[i+len(";base64,"):])
	if err != nil {
		return nil
	}
	return b
}

// handleStatus — klik baris status → buka viewer (overlay statusview).
func (u *UI) handleStatus(gtx layout.Context) {
	for i := range u.statusGroupsCache {
		if i >= len(u.statusClicks) {
			break
		}
		for u.statusClicks[i].Clicked(gtx) {
			u.statusViewIdx = i
			u.overlay = "statusview"
		}
	}
}

// infoData membangun data drawer info dari chat terpilih nyata. nil = demo.
// GetGroupInfo hanya dipanggil saat drawer dibuka (overlay=="info"), bukan tiap frame.
func (u *UI) infoData() *InfoDrawerData {
	if u.core == nil || u.selected == "" {
		return nil
	}
	d := &InfoDrawerData{Name: u.selName, Group: u.selGroup, Sub: u.subtitle}
	if u.selGroup {
		if gi := u.core.GetGroupInfo(u.selected); gi != nil {
			d.Sub = itoa(len(gi.Participants)) + " anggota"
			d.Desc = gi.Topic
		}
	}
	return d
}

// callRows membangun baris pane Panggilan dari log nyata (core.GetCalls). nil =
// mode demo (render standalone). Nama sudah di-resolve ulang di GetCalls.
func (u *UI) callRows() []spCall {
	if u.core == nil {
		return nil
	}
	if u.crCache != nil && time.Since(u.crAt) < time.Second {
		return u.crCache
	}
	cs := u.core.GetCalls()
	out := make([]spCall, 0, len(cs))
	for _, c := range cs {
		out = append(out, spCall{
			name:   c.Name,
			time:   time.Unix(c.TS, 0).Format("15.04"),
			video:  c.Video,
			missed: c.Status == "missed",
		})
	}
	u.crCache, u.crAt = out, time.Now()
	return out
}

// maybeLoadOlder — bila daftar pesan tergulir mendekati ATAS, minta 50 pesan lebih
// lama dari engine (history on-demand WhatsApp; prinsip lazy-history Telegram).
// Throttle per-chat 3s; respons tiba via OnHistorySync ON_DEMAND → GetMessages.
func (u *UI) maybeLoadOlder() {
	if u.core == nil || u.selected == "" || len(u.messages) < 3 {
		return
	}
	if u.msgList.Position.First > 3 { // belum di dekat atas
		return
	}
	u.requestOlder(u.selected)
}

// prefetchHistory — saat buka chat dgn history lokal TIPIS (cuma bootstrap recent),
// tarik satu halaman lama SEGERA → user lihat backfill TERURUT, bukan "cuma pesan
// baru". Constraint WhatsApp: server tak simpan history → minta ke HP; tiru UX
// Telegram = local store permanen + lazy ordered paging.
func (u *UI) prefetchHistory(jid string) {
	if u.core == nil || jid == "" || len(u.messages) >= 25 {
		return
	}
	u.requestOlder(jid)
}

// requestOlder — minta history lama 1× per chat (throttle 3s; respons via
// OnHistorySync ON_DEMAND → GetMessages, terurut sebelum pesan tertua).
func (u *UI) requestOlder(jid string) {
	if u.core == nil || !u.core.HasMoreHistory(jid) { // HP sudah habis → jangan minta
		return
	}
	if u.olderReqChat == jid && time.Since(u.olderReqAt) < 3*time.Second {
		return
	}
	u.olderReqChat, u.olderReqAt = jid, time.Now()
	go u.core.LoadOlderHistory(jid)
}

// ---- percakapan (header + bubble + composer) ----
func (u *UI) conversation(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, u.t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if u.selected == "" {
		return StatesView(gtx, u.th, u.t) // splash + divider demo
	}
	u.maybeLoadOlder() // gulir mendekati atas → minta history lama (lazy, throttled)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.convHeader(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(u.msgClicks) < len(u.messages) { // jamin sebelum index (mid-frame GetMessages)
				u.msgClicks = make([]widget.Clickable, len(u.messages))
			}
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(u.th, &u.msgList).Layout(gtx, len(u.messages), func(gtx layout.Context, i int) layout.Dimensions {
					return u.bubble(gtx, i)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.composer(gtx)
		}),
	)
}

func (u *UI) convHeader(gtx layout.Context) layout.Dimensions {
	for u.headerClick.Clicked(gtx) {
		u.overlay = "info" // klik header → info drawer
	}
	h := gtx.Dp(60)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	u.headerClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: sz} })
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	// ikon overflow header → menu aksi chat yg sedang terbuka (reuse chatctx).
	for u.headMenuClick.Clicked(gtx) {
		for i := range u.chats {
			if u.chats[i].ID == u.selected {
				u.chatCtxChat = u.chats[i] // snapshot chat terbuka
				u.overlay = "chatctx"
				break
			}
		}
	}
	layout.Inset{Left: unit.Dp(18), Right: unit.Dp(8), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, u.selName, u.selected, 40) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 16, u.selName)
						lbl.Color = u.t.Text
						lbl.Font.Weight = font.Medium
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if u.subtitle == "" {
							return layout.Dimensions{}
						}
						col := u.t.Text2 // presence → text2; mengetik/merekam → accent
						if strings.Contains(u.subtitle, "mengetik") || strings.Contains(u.subtitle, "merekam") {
							col = u.t.Accent
						}
						lbl := material.Label(u.th, 12.5, u.subtitle)
						lbl.Color = col
						lbl.MaxLines = 1
						return lbl.Layout(gtx)
					}),
				)
			}),
			// dorong ikon aksi ke kanan
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, nil, "calls") }),  // panggilan (visual)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, nil, "search") }), // cari di chat (visual)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.headMenuClick, "overflow") }),
		)
	})
	return layout.Dimensions{Size: sz}
}

// ---- bubble pesan (.bubble: in/out, RRect, ekor) ----
func (u *UI) bubble(gtx layout.Context, idx int) layout.Dimensions {
	m := u.messages[idx]
	if idx < len(u.msgClicks) {
		for u.msgClicks[idx].Clicked(gtx) {
			switch {
			case m.Type == "voice" && u.OnPlayVoice != nil:
				u.OnPlayVoice(u.selected, m.ID) // tap voice → putar
			case (m.Type == "video" || m.Type == "gif") && u.OnPlayVideo != nil:
				u.OnPlayVideo(u.selected, m.ID, m.Type) // tap video/gif → putar
			default:
				u.ctxIdx, u.ctxMsg = idx, m // snapshot pesan (index bisa bergeser)
				u.overlay = "msgctx"        // klik pesan → context-menu
			}
		}
	}
	out := m.Dir == "out"
	bg := u.t.InBg
	if out {
		bg = u.t.OutBg
	}
	groupIn := u.selGroup && m.Dir == "in"

	maxW := gtx.Constraints.Max.X * 66 / 100
	// susun konten bubble
	content := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxW
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !groupIn || m.Sender == "" {
						return layout.Dimensions{}
					}
					lbl := material.Label(u.th, 13, m.Sender)
					lbl.Color = avatarColor(m.Sender)
					lbl.Font.Weight = font.Bold
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.QuoteName == "" && m.QuoteText == "" {
						return layout.Dimensions{}
					}
					return u.quoteBlock(gtx, m, out) // kutipan pesan dibalas
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.Revoked { // pesan ditarik pengirim → placeholder miring + ikon
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "block", 15, u.t.Text2) }),
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(u.th, 15, "Pesan ini telah dihapus")
								lbl.Color = u.t.Text2
								lbl.Font.Style = font.Italic
								return lbl.Layout(gtx)
							}),
						)
					}
					switch m.Type {
					case "image", "video", "gif":
						return u.mediaThumb(gtx, m) // thumbnail + caption
					case "poll":
						return u.pollBubble(gtx, m) // pertanyaan + opsi
					case "document":
						return u.docBubble(gtx, m)
					case "voice", "audio", "ptt":
						return u.voiceBubble(gtx, m)
					case "location":
						return u.locationBubble(gtx, m)
					case "contact", "vcard":
						return u.contactBubble(gtx, m)
					}
					txt := m.Text
					if txt == "" && m.Type != "" && m.Type != "text" {
						txt = "[" + m.Type + "]"
					}
					if len(m.Mentions) > 0 { // @mention berwarna accent (inline, wrap)
						return u.mentionText(gtx, txt, m.Mentions)
					}
					lbl := material.Label(u.th, 15, txt)
					lbl.Color = u.t.Text
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// .meta: jam + (utk pesan keluar) centang status.
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !m.Edited || m.Revoked {
									return layout.Dimensions{}
								}
								lbl := material.Label(u.th, 11, "Diedit  ")
								lbl.Color = u.t.Text2
								lbl.Font.Style = font.Italic
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(u.th, 11, m.Time)
								lbl.Color = u.t.Text2
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !out {
									return layout.Dimensions{}
								}
								return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return u.statusTick(gtx, m.Status)
								})
							}),
						)
					})
				}),
			)
		})
	}
	// bubble dgn latar RRect + alignment in/out
	align := layout.W
	if out {
		align = layout.E
	}
	bubbleBody := func(gtx layout.Context) layout.Dimensions {
		// rekam konten utk ukur, lalu gambar bg di belakang
		macro := op.Record(gtx.Ops)
		dims := content(gtx)
		call := macro.Stop()
		r := gtx.Dp(18)
		tl, tr := r, r
		if out {
			tr = gtx.Dp(6)
		} else {
			tl = gtx.Dp(6)
		}
		paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: tl, NE: tr, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	colAlign := layout.Start
	if out {
		colAlign = layout.End
	}
	wrap := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return align.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: colAlign}.Layout(gtx,
					layout.Rigid(bubbleBody),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.reactionPills(gtx, m)
					}),
				)
			})
		})
	}
	// Pemisah hari di atas bubble bila ganti tanggal (atau pesan pertama). Bandingkan
	// dgn pesan SEBELUMNYA yg punya Ts>0 (pesan ditarik/sistem bisa Ts==0 → jangan
	// picu pemisah palsu di hari yg sama).
	needSep := false
	if m.Ts > 0 {
		prevDay := int64(-1)
		for j := idx - 1; j >= 0; j-- {
			if u.messages[j].Ts > 0 {
				prevDay = dayKey(u.messages[j].Ts)
				break
			}
		}
		if prevDay < 0 || prevDay != dayKey(m.Ts) {
			needSep = true
		}
	}
	if !needSep {
		return u.msgClicks[idx].Layout(gtx, wrap)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.daySeparator(gtx, dayLabel(m.Ts)) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.msgClicks[idx].Layout(gtx, wrap) }),
	)
}

// statusTick — centang status pesan keluar (✓ sent, ✓✓ delivered/read; biru=read).
func (u *UI) statusTick(gtx layout.Context, status string) layout.Dimensions {
	name := "check"
	col := u.t.Text2
	switch status {
	case "read":
		name, col = "checks", u.t.Tick // biru baca
	case "delivered":
		name = "checks"
	case "sent", "":
		name = "check"
	}
	return icon(gtx, name, 16, col)
}

// daySeparator — pil tanggal di tengah (paritas .day-divider).
func (u *UI) daySeparator(gtx layout.Context, label string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			macro := op.Record(gtx.Ops)
			dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 12, label)
				lbl.Color = u.t.Text2
				return lbl.Layout(gtx)
			})
			call := macro.Stop()
			r := gtx.Dp(8)
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			call.Add(gtx.Ops)
			return dims
		})
	})
}

// dayKey — kunci hari (tahun-bulan-tanggal) dari unix detik utk bandingkan tanggal.
func dayKey(ts int64) int64 {
	if ts <= 0 {
		return 0
	}
	t := time.Unix(ts, 0)
	y, mo, d := t.Date()
	return int64(y)*10000 + int64(mo)*100 + int64(d)
}

// dayLabel — label pemisah: "Hari ini" / "Kemarin" / "2 Jan 2006" (Indonesia).
func dayLabel(ts int64) string {
	if ts <= 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	switch dayKey(ts) {
	case dayKey(now.Unix()):
		return "Hari ini"
	case dayKey(now.AddDate(0, 0, -1).Unix()):
		return "Kemarin"
	}
	bulan := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	y, mo, d := t.Date()
	return itoa(d) + " " + bulan[int(mo)] + " " + itoa(y)
}

// clearReply membatalkan mode balas.
func (u *UI) clearReply() { u.replyTo, u.replyName, u.replyText = "", "", "" }

func (u *UI) composer(gtx layout.Context) layout.Dimensions {
	barH := 0
	if u.replyTo != "" {
		barH = gtx.Dp(46)
	}
	h := gtx.Dp(62) + barH
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Max: image.Pt(sz.X, 1)}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	for u.emojiClick.Clicked(gtx) {
		u.reactMsgID = "" // tombol emoji composer → mode sisip (bukan reaksi pesan)
		u.overlay = "reaction"
	}
	for u.attachClick.Clicked(gtx) {
		u.overlay = "attach" // tombol "+" → menu lampiran
	}
	for u.replyX.Clicked(gtx) {
		u.clearReply()
	}
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if u.replyTo == "" {
				return layout.Dimensions{}
			}
			return u.replyBanner(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(11), Bottom: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.emojiClick, "emoji") }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.attachClick, "plus") }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.composerPill(gtx) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, nil, "mic") }),
				)
			})
		}),
	)
	return layout.Dimensions{Size: sz}
}

// replyBanner — bilah kutipan di atas composer (garis accent + nama + teks + ✕).
func (u *UI) replyBanner(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(12), Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, u.replyName)
							lbl.Color = u.t.Accent
							lbl.Font.Weight = font.Medium
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, u.replyText)
							lbl.Color = u.t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.replyX.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return icon(gtx, "close", 16, u.t.Text2)
						})
					})
				}),
			)
		})
		call := macro.Stop()
		// latar bar + garis accent kiri
		r := gtx.Dp(6)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

func (u *UI) glyphBtn(gtx layout.Context, c *widget.Clickable, iconName string) layout.Dimensions {
	body := func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(40)
		sz := image.Pt(d, d)
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, iconName, 24, u.t.RailIco)
		})
		return layout.Dimensions{Size: sz}
	}
	if c == nil {
		return body(gtx)
	}
	return c.Layout(gtx, body)
}

func (u *UI) composerPill(gtx layout.Context) layout.Dimensions {
	pillH := gtx.Dp(40)
	psz := image.Pt(gtx.Constraints.Max.X, pillH)
	rr := gtx.Dp(22)
	paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: psz}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	gtx.Constraints.Min = psz
	// Kirim saat Enter (Editor.Submit). core nil (demo) → tak kirim.
	for {
		ev, ok := u.editor.Update(gtx)
		if !ok {
			break
		}
		switch ev.(type) {
		case widget.ChangeEvent:
			// indikator "mengetik" keluar: kirim composing sekali saat mulai isi,
			// stop saat kosong (throttle via typingSent agar tak spam socket).
			if u.core != nil && u.selected != "" {
				typing := strings.TrimSpace(u.editor.Text()) != ""
				if typing != u.typingSent {
					u.core.SendTyping(u.selected, typing, false)
					u.typingSent = typing
				}
			}
		case widget.SubmitEvent:
			txt := strings.TrimSpace(u.editor.Text())
			if txt != "" && u.core != nil && u.selected != "" {
				if u.replyTo != "" { // mode balas → kutip pesan
					u.core.Reply(u.selected, txt, u.replyTo, u.replyName, u.replyText)
				} else {
					u.core.SendText(u.selected, txt)
				}
				u.messages = u.core.GetMessages(u.selected)
			}
			if u.core != nil && u.selected != "" && u.typingSent {
				u.core.SendTyping(u.selected, false, false) // berhenti mengetik
				u.typingSent = false
			}
			u.editor.SetText("")
			u.clearReply()
			u.msgList.ScrollTo(len(u.messages)) // setelah kirim → gulir ke bawah
		}
	}
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			ed := material.Editor(u.th, &u.editor, "Ketik pesan")
			ed.Color = u.t.Text
			ed.HintColor = u.t.Text2
			ed.TextSize = 15
			return ed.Layout(gtx)
		})
	})
	return layout.Dimensions{Size: psz}
}

// ---- helpers ----
func initial(name string) string {
	name = strings.TrimSpace(name)
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return strings.ToUpper(string(r))
		}
	}
	return "?"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
