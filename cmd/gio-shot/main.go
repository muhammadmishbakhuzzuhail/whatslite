// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// gio-shot — render UI Gio (internal/gioui) ke PNG secara HEADLESS (EGL
// surfaceless, tanpa display) untuk audit paritas vs Svelte. Data demo statis.
//
//	go run ./cmd/gio-shot [out.png] [w] [h]
//
// Jalankan dgn: LIBGL_ALWAYS_SOFTWARE=1 EGL_PLATFORM=surfaceless go run ./cmd/gio-shot
package main

import (
	"image"
	"image/png"
	"os"
	"strconv"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
)

func main() {
	out := "/tmp/gio_shot.png"
	screen := "main"
	w, h := 1000, 680
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if len(os.Args) > 2 {
		screen = os.Args[2]
	}
	if len(os.Args) > 4 {
		w, _ = strconv.Atoi(os.Args[3])
		h, _ = strconv.Atoi(os.Args[4])
	}

	hw, err := headless.NewWindow(w, h)
	must(err)
	th := material.NewTheme()
	th.Shaper = gioui.NewShaper()
	t := gioui.DarkTheme()
	ui := gioui.NewUI(th, nil) // core nil → data demo
	if os.Getenv("WLGIO_LIGHT") != "" {
		t = gioui.LightTheme()
		ui.SetDark(false)
	}

	draw := func(gtx layout.Context) {
		switch screen {
		case "login":
			// contoh kode QR ala whatsmeow (ref,noise,identity,adv) utk uji render.
			gioui.LoginView(gtx, th, t, "2@abc123XYZ/def456==,Tg9kL+pQr,Zm9vYmFy,YmF6cXV4", nil)
		case "login-phone":
			ed := &widget.Editor{SingleLine: true}
			ed.SetText("628123456789")
			gioui.LoginView(gtx, th, t, "", &gioui.LoginCtl{
				PhoneMode: true, Phone: ed,
				Switch: &widget.Clickable{}, Submit: &widget.Clickable{}, Code: "K7QM-2XPL",
			})
		case "settings":
			gioui.SettingsView(gtx, th, t, &gioui.SettingsCtl{Dark: os.Getenv("WLGIO_LIGHT") == "", Clicks: make([]widget.Clickable, 8)})
		case "bubbles":
			gioui.BubbleTypesView(gtx, th, t)
		case "states":
			gioui.StatesView(gtx, th, t)
		case "convheader":
			gioui.ConvHeaderView(gtx, th, t)
		case "sidepanes":
			gioui.SidePanesView(gtx, th, t, nil)
		case "modals":
			gioui.ModalsView(gtx, th, t, nil)
		case "infodrawer":
			gioui.InfoDrawerView(gtx, th, t, nil)
		case "search":
			gioui.SearchView(gtx, th, t, nil)
		case "picker":
			gioui.PickerView(gtx, th, t, nil)
		case "reactionpicker":
			gioui.ReactionPickerView(gtx, th, t, nil)
		case "composerdetail":
			gioui.ComposerDetailView(gtx, th, t)
		case "gif":
			gioui.GifView(gtx, th, t)
		case "contacts":
			gioui.ContactsPaneView(gtx, th, t, nil, nil, nil)
		case "status":
			gioui.StatusPaneView(gtx, th, t, nil, nil)
		case "channels":
			gioui.ChannelsPaneView(gtx, th, t, nil)
		case "communities":
			gioui.CommunitiesPaneView(gtx, th, t, nil)
		case "bubbleextras":
			gioui.BubbleExtrasView(gtx, th, t)
		case "archived":
			gioui.ArchivedPaneView(gtx, th, t)
		case "scheduled":
			gioui.ScheduledPaneView(gtx, th, t)
		case "msginfo":
			gioui.MsgInfoView(gtx, th, t)
		case "lightbox":
			gioui.LightboxView(gtx, th, t, nil)
		case "starred":
			gioui.StarredPaneView(gtx, th, t, nil)
		case "app-lightbox":
			ui.SetLightbox("m13", "Sunset di pantai 🌅")
			ui.Layout(gtx)
		case "app-quotejump":
			ui.SetHighlight("m3") // sorot pesan asal (hasil lompat dari kutipan)
			ui.Layout(gtx)
		case "app-send":
			ui.SetComposeText("Sampai nanti malam ya!") // composer terisi → tombol kirim
			ui.Layout(gtx)
		case "app-edit":
			ui.SetEditing("m4", "Iya betul, yang deket stasiun") // banner edit pesan
			ui.Layout(gtx)
		case "app-pinned":
			ui.SetPinnedDemo("Sip. Tempatnya yang kemarin kan?", 2) // bar pesan tersemat
			ui.Layout(gtx)
		case "app-unread":
			ui.SetUnreadDemo("m3", 3) // divider "belum dibaca" di atas m3
			ui.Layout(gtx)
		case "app-inchatsearch":
			ui.SetInChatSearch("kemarin") // bilah cari-dalam-chat
			ui.Layout(gtx)
		case "app-settings":
			ui.SetView("settings")
			ui.Layout(gtx)
		case "app-set-profile":
			ui.SetSettingsSub("profile")
			ui.Layout(gtx)
		case "app-set-storage":
			ui.SetSettingsSub("storage")
			ui.Layout(gtx)
		case "app-calls":
			ui.SetView("calls")
			ui.Layout(gtx)
		case "app-splash":
			ui.Deselect()
			ui.Layout(gtx)
		case "app-info":
			ui.SetOverlay("info")
			ui.Layout(gtx)
		case "app-reaction":
			ui.SetOverlay("reaction")
			ui.Layout(gtx)
		case "app-forward":
			ui.SetOverlay("forward")
			ui.Layout(gtx)
		case "app-picker":
			ui.SetOverlay("picker")
			ui.Layout(gtx)
		case "app-msgctx":
			ui.SetOverlay("msgctx")
			ui.Layout(gtx)
		case "app-reply":
			ui.SetReply("Budi Santoso", "Halo! Jadi nanti malam ngumpul jam berapa?")
			ui.Layout(gtx)
		case "app-chatbottom":
			ui.ScrollMessagesToEnd()
			ui.Layout(gtx)
		case "app-newchat":
			ui.SetView("chats")
			ui.Deselect()
			ui.SetSearch("+62 812 3456 7890")
			ui.Layout(gtx)
		case "app-groupcreate":
			ui.SetView("contacts")
			ui.SetOverlay("groupcreate")
			ui.Layout(gtx)
		case "app-lock":
			ui.SetLocked(true)
			ui.Layout(gtx)
		case "app-pinset":
			ui.SetView("settings")
			ui.SetOverlay("pinset")
			ui.Layout(gtx)
		case "app-attach":
			ui.SetOverlay("attach")
			ui.Layout(gtx)
		case "app-loccompose":
			ui.SetOverlay("loccompose")
			ui.Layout(gtx)
		case "app-pollcompose":
			ui.SetOverlay("pollcompose")
			ui.Layout(gtx)
		case "app-chatctx":
			ui.SetOverlay("chatctx")
			ui.Layout(gtx)
		default:
			ui.Layout(gtx)
		}
	}

	ops := new(op.Ops)
	// dua frame: frame-1 memicu refresh() (load data demo), frame-2 menggambarnya.
	for i := 0; i < 2; i++ {
		ops.Reset()
		gtx := layout.Context{Ops: ops, Constraints: layout.Exact(image.Pt(w, h)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		draw(gtx)
		must(hw.Frame(ops))
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	must(hw.Screenshot(img))
	f, err := os.Create(out)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
	println("gio → " + out)
}

func must(err error) {
	if err != nil {
		println("gio-shot ERR:", err.Error())
		os.Exit(1)
	}
}
