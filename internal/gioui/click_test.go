// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// click_test.go — bukti audit INTERAKTIF (bukan statis): suntik event pointer
// headless via input.Router → verifikasi handler tombol benar-benar terpicu.
// Menjawab "apakah bisa?" — ya, klik bisa diuji tanpa layar.
package gioui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// clickAt — render `w` penuh-area, suntik Press+Release di pos, lalu laporkan
// apakah Clickable terpicu pada frame berikutnya. Harness umum untuk uji klik.
func clickAt(r *input.Router, ops *op.Ops, sz image.Point, btn *widget.Clickable, pos f32.Point) bool {
	full := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Max} }
	// frame 1: daftarkan handler pointer area.
	ops.Reset()
	gtx := layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(sz)}
	btn.Layout(gtx, full)
	r.Frame(ops)
	// suntik klik (tekan + lepas) di posisi.
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos},
	)
	// frame 2: proses event → Clicked.
	ops.Reset()
	gtx = layout.Context{Ops: ops, Source: r.Source(), Constraints: layout.Exact(sz)}
	clicked := false
	for btn.Clicked(gtx) {
		clicked = true
	}
	btn.Layout(gtx, full)
	r.Frame(ops)
	return clicked
}

// TestHeadlessClickFires — bukti dasar: klik headless memicu Clickable.Clicked.
func TestHeadlessClickFires(t *testing.T) {
	var r input.Router
	var btn widget.Clickable
	ops := new(op.Ops)
	sz := image.Pt(120, 48)
	if !clickAt(&r, ops, sz, &btn, f32.Pt(60, 24)) {
		t.Fatal("klik headless di dalam area TIDAK memicu Clickable.Clicked")
	}
}

// TestHeadlessClickMiss — klik di LUAR area tak boleh memicu (kontrol negatif).
func TestHeadlessClickMiss(t *testing.T) {
	var r input.Router
	var btn widget.Clickable
	ops := new(op.Ops)
	sz := image.Pt(120, 48)
	// area handler 120x48, tapi widget penuh = seluruh area; untuk miss, pakai
	// posisi negatif (di luar konstrain) → tak ada hit.
	if clickAt(&r, ops, sz, &btn, f32.Pt(-5, -5)) {
		t.Fatal("klik di luar area SALAH memicu Clickable.Clicked")
	}
}

// TestRailClickChangesView — audit interaktif NYATA: render UI penuh, klik tombol
// rail "status", verifikasi u.view berpindah. Membuktikan tombol asli terpicu.
func TestRailClickChangesView(t *testing.T) {
	u := NewUI(material.NewTheme(), nil) // mode demo (tanpa engine)
	var r input.Router
	ops := new(op.Ops)
	sz := image.Pt(1000, 680)
	frame := func() {
		ops.Reset()
		gtx := layout.Context{
			Ops: ops, Source: r.Source(),
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(sz),
		}
		u.Layout(gtx)
		r.Frame(ops)
	}
	frame() // muat data demo + daftar handler
	frame()
	if u.view != "chats" {
		t.Fatalf("view awal = %q, harusnya chats", u.view)
	}
	// rail: lebar 56 (x center 28), tombol di bawah titlebar(34): y = 34+14+i*50+22.
	// i=1 → "status".
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(28, 120)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(28, 120)},
	)
	frame() // proses klik → handler railBtn ubah view
	if u.view != "status" {
		t.Fatalf("setelah klik rail status, view = %q (harusnya status)", u.view)
	}
}
