// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// ui.go — tata letak 3-panel (rail | sidebar | percakapan), daftar chat, bubble
// pesan, avatar. Menggambar bentuk kustom (RRect bubble, lingkaran avatar) via
// clip — membuktikan Gio bisa desain pixel-WhatsApp. Data dari engine in-process.
package gioui

import (
	"image"
	"image/color"
	"sort"
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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

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
	shown        []int // indeks u.chats yg lolos filter+pencarian (urut tampil)

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

	// menu aksi baris chat (klik-kanan): target + item.
	chatCtxIdx    int
	chatCtxItems  [5]widget.Clickable
	headMenuClick widget.Clickable // ikon overflow header → menu chat terbuka

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
	ctxIdx      int                 // index pesan utk context-menu
	ctxItems    [6]widget.Clickable // item menu (react/reply/forward/star/info/delete)

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
	u.rpClicks = make([]widget.Clickable, len(RpEmoji()))
	u.photos = map[string]paint.ImageOp{}
	u.photoTried = map[string]bool{}
	u.media = map[string]paint.ImageOp{}
	u.mediaTried = map[string]bool{}
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
	for u.setClicks[0].Clicked(gtx) { // Tema
		u.dark = !u.dark
		u.t = newTheme(u.dark)
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
		InfoDrawerView(gtx, u.th, u.t)
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
	}
}

// doCtxAction menjalankan aksi context-menu pesan terhadap engine. Bintangi/Hapus
// langsung; Balas mengaktifkan banner balas di composer (kirim → core.Reply).
func (u *UI) doCtxAction(label string) {
	if u.ctxIdx < 0 || u.ctxIdx >= len(u.messages) {
		return
	}
	m := u.messages[u.ctxIdx]
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
			if u.OnAttach != nil && u.selected != "" {
				u.OnAttach(u.selected, AttachCategory(i))
			}
			u.overlay = ""
		}
	}
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
	if u.chatCtxIdx < 0 || u.chatCtxIdx >= len(u.chats) {
		u.overlay = ""
		return layout.Dimensions{}
	}
	c := u.chats[u.chatCtxIdx]
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
		return SettingsView(gtx, u.th, u.t, &SettingsCtl{Dark: u.dark, Clicks: u.setClicks[:]})
	case "calls":
		return SidePanesView(gtx, u.th, u.t, u.callRows())
	case "contacts":
		return ContactsPaneView(gtx, u.th, u.t, u.contactGroups())
	case "status":
		return StatusPaneView(gtx, u.th, u.t, u.statusRows())
	case "channels":
		return ChannelsPaneView(gtx, u.th, u.t, u.channelRows())
	}
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Rect{Max: sz}.Op())

	u.handleChatFilter(gtx)
	u.computeShown()
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
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(u.th, &u.chatList).Layout(gtx, len(u.shown), func(gtx layout.Context, i int) layout.Dimensions {
				return u.chatRow(gtx, u.shown[i])
			})
		}),
	)
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
				u.chatCtxIdx = i
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
	cs := u.core.GetContacts()
	sort.Slice(cs, func(i, j int) bool {
		return strings.ToLower(cs[i].Name) < strings.ToLower(cs[j].Name)
	})
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
		cur.items = append(cur.items, cpContact{name: c.Name, about: c.Phone})
	}
	return groups
}

// channelRows membangun pane Saluran dari saluran nyata (core.GetChannels). nil = demo.
func (u *UI) channelRows() []chnChannel {
	if u.core == nil {
		return nil
	}
	cs := u.core.GetChannels()
	out := make([]chnChannel, 0, len(cs))
	for _, c := range cs {
		out = append(out, chnChannel{name: c.Name, subs: fmtSubs(c.Subscribers)})
	}
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
	gs := u.core.GetStatuses()
	out := make([]stpItem, 0, len(gs))
	for _, g := range gs {
		if g.Mine {
			continue // status sendiri tampil di baris My-status, bukan daftar
		}
		out = append(out, stpItem{name: g.Name, time: g.Time, seen: false})
	}
	return out
}

// callRows membangun baris pane Panggilan dari log nyata (core.GetCalls). nil =
// mode demo (render standalone). Nama sudah di-resolve ulang di GetCalls.
func (u *UI) callRows() []spCall {
	if u.core == nil {
		return nil
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
	return out
}

// maybeLoadOlder — bila daftar pesan tergulir mendekati ATAS, minta 50 pesan lebih
// lama dari engine (history on-demand WhatsApp; prinsip lazy-history Telegram).
// Throttle per-chat 3s; respons tiba via OnHistorySync ON_DEMAND → GetMessages.
func (u *UI) maybeLoadOlder() {
	if u.core == nil || u.selected == "" || len(u.messages) < 15 {
		return
	}
	if u.msgList.Position.First > 3 { // belum di dekat atas
		return
	}
	if u.olderReqChat == u.selected && time.Since(u.olderReqAt) < 3*time.Second {
		return // baru saja minta utk chat ini
	}
	u.olderReqChat, u.olderReqAt = u.selected, time.Now()
	chat := u.selected
	go u.core.LoadOlderHistory(chat)
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
				u.chatCtxIdx = i
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
				u.ctxIdx = idx
				u.overlay = "msgctx" // klik pesan → context-menu
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
					}
					txt := m.Text
					if txt == "" && m.Type != "" && m.Type != "text" {
						txt = "[" + m.Type + "]"
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
	// Pemisah hari di atas bubble bila ganti tanggal (atau pesan pertama).
	needSep := false
	if m.Ts > 0 {
		if idx == 0 || (idx-1 < len(u.messages) && dayKey(u.messages[idx-1].Ts) != dayKey(m.Ts)) {
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
