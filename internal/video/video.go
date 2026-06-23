// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// video.go — putar video pesan via libmpv (cgo) IN-PROCESS. v1: libmpv membuka
// window-nya sendiri (sederhana, jalan). Embed-ke-bubble (render-to-GL-texture
// via mpv render API) = follow-up yang lebih rumit/rapuh.
//
// cgo butuh: libmpv (pkg-config mpv). Sudah ada (2.5.0).
package video

/*
#cgo pkg-config: mpv
#include <mpv/client.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"os"
	"sync"
	"unsafe"
)

// Player membungkus satu konteks libmpv.
type Player struct {
	mu  sync.Mutex
	ctx *C.mpv_handle
	tmp string
}

// PlayBytes: tulis byte video ke file sementara → libmpv loadfile (window mpv
// muncul + putar). Kembalikan Player utk stop/cleanup.
func PlayBytes(data []byte, ext string) (*Player, error) {
	if len(data) == 0 {
		return nil, errors.New("video: data kosong")
	}
	f, err := os.CreateTemp("", "wlvid-*"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	f.Close()
	p, err := PlayFile(f.Name())
	if err != nil {
		os.Remove(f.Name())
		return nil, err
	}
	p.tmp = f.Name()
	return p, nil
}

// PlayAudioOnly: putar HANYA audio dari byte video (vid=no → TANPA window mpv).
// Dipakai utk video status inline (frame digambar Gio, suara dari sini).
func PlayAudioOnly(data []byte, ext string) (*Player, error) {
	if len(data) == 0 {
		return nil, errors.New("video: data kosong")
	}
	f, err := os.CreateTemp("", "wlaud-*"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	f.Close()
	ctx := C.mpv_create()
	if ctx == nil {
		os.Remove(f.Name())
		return nil, errors.New("video: mpv_create gagal")
	}
	setOpt := func(k, v string) {
		ck, cv := C.CString(k), C.CString(v)
		C.mpv_set_option_string(ctx, ck, cv)
		C.free(unsafe.Pointer(ck))
		C.free(unsafe.Pointer(cv))
	}
	setOpt("vid", "no")     // tanpa video → tak buka window
	setOpt("force-window", "no")
	if rc := C.mpv_initialize(ctx); rc < 0 {
		C.mpv_terminate_destroy(ctx)
		os.Remove(f.Name())
		return nil, errors.New("video: mpv_initialize gagal")
	}
	cpath := C.CString(f.Name())
	defer C.free(unsafe.Pointer(cpath))
	cload := C.CString("loadfile")
	defer C.free(unsafe.Pointer(cload))
	args := []*C.char{cload, cpath, nil}
	if rc := C.mpv_command(ctx, &args[0]); rc < 0 {
		C.mpv_terminate_destroy(ctx)
		os.Remove(f.Name())
		return nil, errors.New("video: loadfile gagal")
	}
	return &Player{ctx: ctx, tmp: f.Name()}, nil
}

// SetPause menjeda/melanjutkan pemutaran (sinkron dgn pause viewer).
func (p *Player) SetPause(pause bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx == nil {
		return
	}
	v := "no"
	if pause {
		v = "yes"
	}
	ck, cv := C.CString("pause"), C.CString(v)
	C.mpv_set_property_string(p.ctx, ck, cv)
	C.free(unsafe.Pointer(ck))
	C.free(unsafe.Pointer(cv))
}

// PlayFile: putar file video lewat libmpv.
func PlayFile(path string) (*Player, error) {
	ctx := C.mpv_create()
	if ctx == nil {
		return nil, errors.New("video: mpv_create gagal")
	}
	if rc := C.mpv_initialize(ctx); rc < 0 {
		C.mpv_terminate_destroy(ctx)
		return nil, errors.New("video: mpv_initialize gagal")
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	cload := C.CString("loadfile")
	defer C.free(unsafe.Pointer(cload))
	// mpv_command(ctx, {"loadfile", path, NULL})
	args := []*C.char{cload, cpath, nil}
	if rc := C.mpv_command(ctx, &args[0]); rc < 0 {
		C.mpv_terminate_destroy(ctx)
		return nil, errors.New("video: loadfile gagal")
	}
	return &Player{ctx: ctx}, nil
}

// Stop menutup player + hapus file sementara.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx != nil {
		C.mpv_terminate_destroy(p.ctx)
		p.ctx = nil
	}
	if p.tmp != "" {
		os.Remove(p.tmp)
		p.tmp = ""
	}
}
