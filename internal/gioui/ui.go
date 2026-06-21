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
	"strings"
	"time"
	"unicode"

	"gioui.org/font"
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
	th    *material.Theme
	core  *app.App
	t     Theme
	dark  bool
	state string
	qr    string // kode QR pairing mentah terbaru (dari core.QRCode); "" = belum ada
	view  string // pane sidebar aktif: chats|calls|settings

	// alur login via nomor telepon (alternatif QR): toggle, input, kode 8-karakter.
	loginPhone  bool
	phoneEd     widget.Editor
	loginSwitch widget.Clickable
	loginSubmit widget.Clickable
	pairCode    string

	setClicks [8]widget.Clickable // baris pane setelan (0=Tema … 7=Keluar)

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
	photos     map[string]paint.ImageOp // foto avatar in-memory (jid/nama → op)

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
}

// ctxMenu = item context-menu pesan (glyph + aksi/overlay tujuan).
var ctxMenu = []struct{ icon, label, to string }{
	{"emoji", "Reaksi", "reaction"}, {"reply", "Balas", ""}, {"forward", "Teruskan", "forward"},
	{"star", "Bintangi", ""}, {"info", "Info", "msginfo"}, {"trash", "Hapus", ""},
}

// SetOverlay: utk render-tool menguji popup headless.
func (u *UI) SetOverlay(o string) { u.overlay = o }

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
	u.photos = map[string]paint.ImageOp{}
	if core == nil { // demo: foto sintetis utk membuktikan avatar-foto bulat
		u.photos["Andi Pratama"] = synthPhoto()
	}
	return u
}

// SetDark: ganti tema (dipakai render-tool utk audit light/dark).
func (u *UI) SetDark(d bool) { u.dark = d; u.t = newTheme(d) }

// SetView/Deselect: utk render-tool menguji state navigasi headless.
func (u *UI) SetView(v string) { u.view = v }
func (u *UI) Deselect()        { u.selected = "" }

func (u *UI) refresh() {
	if u.core == nil { // mode demo: data statis (uji render tanpa engine/jaringan)
		u.chats = demoChats()
		if u.selected == "" {
			u.selected, u.selName, u.selGroup = "2", "Keluarga", true
		}
		u.messages = demoMessages()
	} else {
		u.state = u.core.GetState()
		u.qr = u.core.QRCode()
		u.chats = u.core.GetChats()
		if u.selected != "" {
			u.messages = u.core.GetMessages(u.selected)
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
	return []app.MessageDTO{
		{ID: "m1", Dir: "in", Type: "text", Text: "Halo! Jadi nanti malam ngumpul jam berapa?", Time: "19.02", Sender: "Budi Santoso"},
		{ID: "m2", Dir: "out", Type: "text", Text: "Jam 8 ya, di tempat biasa 👍", Time: "19.03", Status: "read"},
		{ID: "m3", Dir: "in", Type: "text", Text: "Oke sip. Aku bawa kamera sekalian foto-foto.", Time: "19.04", Sender: "Citra Dewi"},
		{ID: "m4", Dir: "out", Type: "text", Text: "Mantap! Sampai nanti 🙌", Time: "19.05", Status: "read"},
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
		ModalsView(gtx, u.th, u.t)
	case "msginfo":
		MsgInfoView(gtx, u.th, u.t)
	case "reaction":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X, gtx.Constraints.Max.Y = gtx.Dp(352), gtx.Dp(400)
			return ReactionPickerView(gtx, u.th, u.t)
		})
	case "lightbox":
		LightboxView(gtx, u.th, u.t)
	case "picker":
		PickerView(gtx, u.th, u.t)
	case "msgctx":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(220), gtx.Dp(220)
			return u.ctxMenuView(gtx)
		})
	}
}

// ctxMenuView — menu aksi pesan (.menu): kartu bg + baris glyph+label klik.
func (u *UI) ctxMenuView(gtx layout.Context) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(ctxMenu))
	for i := range ctxMenu {
		i := i
		it := ctxMenu[i]
		for u.ctxItems[i].Clicked(gtx) {
			u.overlay = it.to // pindah ke popup tujuan ("" = tutup)
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
		return SidePanesView(gtx, u.th, u.t)
	case "contacts":
		return ContactsPaneView(gtx, u.th, u.t)
	case "status":
		return StatusPaneView(gtx, u.th, u.t)
	case "channels":
		return ChannelsPaneView(gtx, u.th, u.t)
	}
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Rect{Max: sz}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.header(gtx, w, "Chat", u.t.Text, 23, font.Bold)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(u.th, &u.chatList).Layout(gtx, len(u.chats), func(gtx layout.Context, i int) layout.Dimensions {
				return u.chatRow(gtx, i)
			})
		}),
	)
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
		}
		// bg hover/active
		dims := layout.UniformInset(0).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// .chat-list pad 4/8 + .chat-row pad 10/12 → vert 10, horiz 8+12=20.
			return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.avatar(gtx, c.Name, 49)
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
func (u *UI) avatar(gtx layout.Context, name string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	// Foto in-memory (byte engine → ImageOp) di-mask bulat; else inisial.
	if ph, ok := u.photos[name]; ok {
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

// ---- percakapan (header + bubble + composer) ----
func (u *UI) conversation(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, u.t.Wallpaper, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if u.selected == "" {
		return StatesView(gtx, u.th, u.t) // splash + divider demo
	}
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
	layout.Inset{Left: unit.Dp(18), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, u.selName, 40) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 16, u.selName)
				lbl.Color = u.t.Text
				lbl.Font.Weight = font.Medium
				return lbl.Layout(gtx)
			}),
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
					txt := m.Text
					if txt == "" {
						txt = "[" + m.Type + "]"
					}
					lbl := material.Label(u.th, 15, txt)
					lbl.Color = u.t.Text
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 11, m.Time)
					lbl.Color = u.t.Text2
					return layout.E.Layout(gtx, lbl.Layout)
				}),
			)
		})
	}
	// bubble dgn latar RRect + alignment in/out
	align := layout.W
	if out {
		align = layout.E
	}
	wrap := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return align.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
			})
		})
	}
	return u.msgClicks[idx].Layout(gtx, wrap)
}

func (u *UI) composer(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(62)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Max: image.Pt(sz.X, 1)}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	for u.emojiClick.Clicked(gtx) {
		u.overlay = "reaction"
	}
	for u.attachClick.Clicked(gtx) {
		u.overlay = "picker"
	}
	layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(11), Bottom: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
	return layout.Dimensions{Size: sz}
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
		if _, ok := ev.(widget.SubmitEvent); ok {
			txt := strings.TrimSpace(u.editor.Text())
			if txt != "" && u.core != nil && u.selected != "" {
				u.core.SendText(u.selected, txt)
				u.messages = u.core.GetMessages(u.selected)
			}
			u.editor.SetText("")
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
