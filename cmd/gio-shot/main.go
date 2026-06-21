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
			gioui.SearchView(gtx, th, t)
		case "picker":
			gioui.PickerView(gtx, th, t)
		case "reactionpicker":
			gioui.ReactionPickerView(gtx, th, t, nil)
		case "composerdetail":
			gioui.ComposerDetailView(gtx, th, t)
		case "gif":
			gioui.GifView(gtx, th, t)
		case "contacts":
			gioui.ContactsPaneView(gtx, th, t, nil)
		case "status":
			gioui.StatusPaneView(gtx, th, t, nil, nil)
		case "channels":
			gioui.ChannelsPaneView(gtx, th, t, nil)
		case "bubbleextras":
			gioui.BubbleExtrasView(gtx, th, t)
		case "archived":
			gioui.ArchivedPaneView(gtx, th, t)
		case "scheduled":
			gioui.ScheduledPaneView(gtx, th, t)
		case "msginfo":
			gioui.MsgInfoView(gtx, th, t)
		case "lightbox":
			gioui.LightboxView(gtx, th, t)
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
		case "app-attach":
			ui.SetOverlay("attach")
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
