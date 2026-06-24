// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// whatslite-gio — UI WhatsLite native pure-Go (Gio) yang memanggil engine
// whatsmeow IN-PROCESS (tanpa jembatan IPC/HTTP). UI di internal/gioui agar
// bisa dipakai ulang oleh render-tool (cmd/gio-shot) untuk audit headless.
package main

import (
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	gioapp "gioui.org/app"
	"gioui.org/gpu/headless"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/gioui"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/video"
	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/voice"
)

func main() {
	// Profiling memori opt-in (mati secara default; hanya aktif bila env diset).
	// WLGIO_PPROF=1 → server pprof di localhost:6060 (go tool pprof / /debug/pprof).
	if os.Getenv("WLGIO_PPROF") != "" {
		go func() { _ = http.ListenAndServe("localhost:6060", nil) }()
	}
	// WLGIO_MEMLOG=1 → log statistik memori runtime tiap 5 detik (heap/sys/GC/goroutine).
	if os.Getenv("WLGIO_MEMLOG") != "" {
		go func() {
			var m runtime.MemStats
			for range time.NewTicker(5 * time.Second).C {
				runtime.ReadMemStats(&m)
				log.Printf("[mem] heapAlloc=%.1fMB sys=%.1fMB numGC=%d goroutines=%d",
					float64(m.HeapAlloc)/1e6, float64(m.Sys)/1e6, m.NumGC, runtime.NumGoroutine())
			}
		}()
	}

	var core *app.App
	if os.Getenv("WLGIO_DEMO") == "" {
		core = app.NewApp()
		if err := core.StartupHeadless(context.Background()); err != nil {
			log.Fatal("[gio] startup engine: ", err)
		}
		core.Connect()
	}

	// Shutdown bersih saat sinyal (run.sh pakai pkill = SIGTERM). Tanpa ini, app
	// mati abrupt → whatsmeow tak Disconnect: bila terbunuh di antara ratchet-maju
	// dan ack ke server, pesan yg di-resend GAGAL DIDEKRIPSI. Disconnect rapi
	// mencegah desync sesi Signal tsb.
	shutdown := func() {
		if core != nil {
			core.Shutdown(context.Background())
		}
	}
	if core != nil {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			shutdown()
			os.Exit(0)
		}()
	}

	// app-id Wayland / kelas X11 — default-nya nama biner ("whatslite-gio") yg muncul
	// di panel atas GNOME. Set ke "WhatsLite" agar nama app benar di taskbar/panel.
	gioapp.ID = "WhatsLite"

	go func() {
		w := new(gioapp.Window)
		// Decorated(false): gambar titlebar sendiri (CSD) → tampilan gelap konsisten
		// (Wayland: ganti libdecor/GTK bar). Pada X11 Gio mengabaikan ini (WM tetap).
		w.Option(gioapp.Title("WhatsLite"), gioapp.Size(unit.Dp(1000), unit.Dp(680)),
			gioapp.MinSize(unit.Dp(720), unit.Dp(480)), gioapp.Decorated(false))
		err := run(w, core)
		shutdown() // tutup window → Disconnect + tutup DB sebelum keluar
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	gioapp.Main()
}

func run(w *gioapp.Window, core *app.App) error {
	th := material.NewTheme()
	th.Shaper = gioui.NewShaper()
	ui := gioui.NewUI(th, core)
	// Voice note (ogg-opus) in-process: byte engine → internal/voice (cgo libopus).
	if core != nil {
		ui.OnPlayVoice = func(chat, id string) {
			if b := core.MediaBytes(chat, id); len(b) > 0 {
				go func() { _, _ = voice.Play(b) }()
			}
		}
		ui.OnPlayVideo = func(chat, id, typ string) {
			if b := core.MediaBytes(chat, id); len(b) > 0 {
				ext := ".mp4"
				if typ == "gif" {
					ext = ".gif"
				}
				go func() { _, _ = video.PlayBytes(b, ext) }()
			}
		}
	}

	// Lampiran: dialog berkas native (x/explorer, pure-Go) → SendMedia in-process.
	// Hanya kategori berbasis berkas (media/document); contact/location/poll = TODO.
	expl := explorer.NewExplorer(w)
	if core != nil {
		ui.OnAttach = func(chat, category string) {
			go pickAndSend(expl, core, ui, chat, category)
		}
		ui.OnSaveMedia = func(chat, id, name string) {
			go saveMedia(expl, core, chat, id, name)
		}
		ui.OnStatusMedia = func() {
			go pickAndPostStatus(expl, core, ui)
		}
		ui.OnOpenURL = func(url string) {
			if url != "" {
				_ = exec.Command("xdg-open", url).Start()
			}
		}
		ui.OnSetPhoto = func() { go pickAndSetPhoto(expl, core) } // ganti foto profil
		// poster still utk byte yg tak bisa image.Decode: GIF WA (mp4), stiker
		// webp animasi, video → frame pertama via ffmpeg.
		ui.OnMediaPoster = func(data []byte, ext string) image.Image {
			img, err := video.FirstFrame(data, ext, 320)
			if err != nil || img == nil {
				return nil
			}
			return img
		}
		ui.OnStatusVideo = func(id string) gioui.StatusVideo {
			b := core.MediaBytes("status@broadcast", id)
			if len(b) == 0 {
				return nil
			}
			fs, err := video.OpenFrames(b, ".mp4", 480)
			if err != nil {
				return nil
			}
			au, _ := video.PlayAudioOnly(b, ".mp4") // suara (best-effort, no window)
			return &statusVideoSession{fs: fs, au: au}
		}
	}
	ui.OnWinAction = func(a string) { // titlebar custom → aksi window (CSD)
		switch a {
		case "minimize":
			w.Perform(system.ActionMinimize)
		case "maximize":
			w.Perform(system.ActionMaximize)
		case "unmaximize":
			w.Perform(system.ActionUnmaximize)
		case "close":
			w.Perform(system.ActionClose)
		}
	}

	go func() {
		last := -1
		for range time.NewTicker(700 * time.Millisecond).C {
			if core != nil { // judul window dgn badge belum-dibaca ("(3) WhatsLite")
				if n := core.UnreadTotal(); n != last {
					last = n
					w.Option(gioapp.Title(titleFor(n)))
				}
			}
			w.Invalidate()
		}
	}()

	// Helper auto-capture: WLGIO_SHOTDIR set → simpan PNG frame LIVE (data WA asli)
	// tiap WLGIO_SHOT_EVERY detik (default 3) utk loop analisa→perbaikan UI/UX.
	// Render headless dgn ops yg SAMA spt window (satu goroutine → tanpa race).
	shot := newShooter()

	var ops op.Ops
	for {
		evt := w.Event()
		expl.ListenEvents(evt) // x/explorer perlu lihat tiap event window
		switch e := evt.(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			ui.Layout(gtx)
			shot.maybeCapture(gtx.Ops, e.Size) // foto seluruh frame live
			shot.maybeCaptureScreens(ui, e)    // atau foto tiap layar bernama
			e.Frame(gtx.Ops)
		}
	}
}

// titleFor — judul window dgn badge belum-dibaca (ala WhatsApp Web).
func titleFor(unread int) string {
	switch {
	case unread <= 0:
		return "WhatsLite"
	case unread > 99:
		return "(99+) WhatsLite"
	default:
		return "(" + strconv.Itoa(unread) + ") WhatsLite"
	}
}

// pickAndSend membuka dialog berkas, baca byte, deteksi mime → kind, lalu kirim
// via core.SendMedia (data-URI base64, in-process). category: media|document.
func pickAndSend(expl *explorer.Explorer, core *app.App, ui *gioui.UI, chat, category string) {
	if category == "camera" { // tangkap foto webcam (ffmpeg) → pratinjau, bukan dialog
		if uri := core.CapturePhoto(); uri != "" {
			ui.SetPendingMedia("image", uri)
		}
		return
	}
	var exts []string
	switch category {
	case "media":
		exts = []string{"jpg", "jpeg", "png", "gif", "webp", "mp4", "mov", "webm"}
	case "document":
		exts = nil // semua jenis
	default:
		return // contact/location/poll: butuh dialog input sendiri (TODO)
	}
	rc, err := expl.ChooseFile(exts...)
	if err != nil || rc == nil {
		return
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil || len(data) == 0 {
		return
	}
	mime := http.DetectContentType(data)
	kind := "document"
	switch {
	case category == "document":
		kind = "document"
	case strings.HasPrefix(mime, "image/"):
		kind = "image"
	case strings.HasPrefix(mime, "video/"):
		kind = "video"
	}
	uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	name := "" // nama berkas asli (utk rename dokumen di pratinjau)
	if f, ok := rc.(interface{ Name() string }); ok {
		name = filepath.Base(f.Name())
	}
	// Semua kategori (termasuk dokumen) → pratinjau dulu: caption / rename / info.
	ui.SetPendingMediaNamed(kind, name, uri)
}

// pickAndSetPhoto — dialog berkas (gambar) → data-URI → SetMyPhoto (foto profil).
func pickAndSetPhoto(expl *explorer.Explorer, core *app.App) {
	rc, err := expl.ChooseFile("jpg", "jpeg", "png", "webp")
	if err != nil || rc == nil {
		return
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil || len(data) == 0 {
		return
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return
	}
	uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	core.SetMyPhoto(uri, uri) // full + preview (engine resize bila perlu)
}

// statusVideoSession — adapter gioui.StatusVideo: frame (ffmpeg) + audio (libmpv
// vid=no). Memenuhi interface tanpa menyeret cgo ke paket gioui.
type statusVideoSession struct {
	fs *video.FrameStream
	au *video.Player
}

func (s *statusVideoSession) Frame(el time.Duration) (image.Image, bool) {
	fr, ended := s.fs.Frame(el)
	if fr == nil {
		return nil, ended
	}
	return fr, ended
}
func (s *statusVideoSession) Duration() time.Duration { return s.fs.Duration() }
func (s *statusVideoSession) SetPause(p bool) {
	if s.au != nil {
		s.au.SetPause(p)
	}
}
func (s *statusVideoSession) Close() {
	s.fs.Close()
	if s.au != nil {
		s.au.Stop()
	}
}

// pickAndPostStatus — dialog berkas (foto/video) → unggah sbg STATUS sendiri via
// core.PostMediaStatus (data-URI). Tutup composer setelah terkirim.
func pickAndPostStatus(expl *explorer.Explorer, core *app.App, ui *gioui.UI) {
	rc, err := expl.ChooseFile("jpg", "jpeg", "png", "webp", "mp4", "mov", "webm")
	if err != nil || rc == nil {
		return
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil || len(data) == 0 {
		return
	}
	mime := http.DetectContentType(data)
	kind := "image"
	if strings.HasPrefix(mime, "video/") {
		kind = "video"
	}
	uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	core.PostMediaStatus(kind, "", uri)
	ui.CloseStatusCompose() // kembali ke pane status
}

// saveMedia membuka dialog "simpan sebagai" native (x/explorer), lalu menulis byte
// media penuh (core.MediaBytes) ke berkas pilihan. name = saran nama awal.
func saveMedia(expl *explorer.Explorer, core *app.App, chat, id, name string) {
	b := core.MediaBytes(chat, id)
	if len(b) == 0 {
		return
	}
	wc, err := expl.CreateFile(name)
	if err != nil || wc == nil {
		return
	}
	defer wc.Close()
	_, _ = wc.Write(b)
}

// shooter mengambil PNG frame aplikasi yg sedang berjalan (data nyata) ke
// WLGIO_SHOTDIR. Render headless pakai ops yg sama → identik dgn yg di layar.
type shooter struct {
	dir     string
	every   time.Duration
	last    time.Time
	hw      *headless.Window
	size    image.Point
	n       int
	screens []string // WLGIO_SHOT_SCREENS: potret tiap layar bernama (data nyata)
}

func newShooter() *shooter {
	dir := os.Getenv("WLGIO_SHOTDIR")
	if dir == "" {
		return &shooter{}
	}
	every := 3 * time.Second
	if v := os.Getenv("WLGIO_SHOT_EVERY"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			every = time.Duration(sec) * time.Second
		}
	}
	var screens []string
	if v := os.Getenv("WLGIO_SHOT_SCREENS"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				screens = append(screens, s)
			}
		}
	}
	_ = os.MkdirAll(dir, 0o755)
	if len(screens) > 0 {
		log.Printf("[gio] auto-capture layar %v → %s tiap %s", screens, dir, every)
	} else {
		log.Printf("[gio] auto-capture aktif → %s tiap %s", dir, every)
	}
	return &shooter{dir: dir, every: every, screens: screens}
}

// ensureHW (re)membuat window headless seukuran window saat ini.
func (s *shooter) ensureHW(size image.Point) bool {
	if s.hw != nil && s.size == size {
		return true
	}
	if s.hw != nil {
		s.hw.Release()
	}
	hw, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		log.Printf("[gio] capture: headless gagal: %v", err)
		s.dir = "" // matikan agar tak spam error
		return false
	}
	s.hw, s.size = hw, size
	return true
}

// snap render ops → PNG di WLGIO_SHOTDIR (label = nama berkas tanpa ekstensi).
func (s *shooter) snap(ops *op.Ops, size image.Point, label string) {
	if err := s.hw.Frame(ops); err != nil {
		return
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := s.hw.Screenshot(img); err != nil {
		return
	}
	name := filepath.Join(s.dir, "wlive-"+label+".png")
	f, err := os.Create(name)
	if err != nil {
		return
	}
	defer f.Close()
	if png.Encode(f, img) == nil {
		s.n++
		log.Printf("[gio] capture #%d → %s", s.n, name)
	}
}

// maybeCapture memotret SELURUH frame live (default, ops yg sama spt window).
// Dilewati bila mode layar-bernama aktif (lihat maybeCaptureScreens).
func (s *shooter) maybeCapture(ops *op.Ops, size image.Point) {
	if s.dir == "" || len(s.screens) > 0 || size.X <= 0 || size.Y <= 0 || time.Since(s.last) < s.every {
		return
	}
	s.last = time.Now()
	if !s.ensureHW(size) {
		return
	}
	s.snap(ops, size, s.last.UTC().Format("20060102-150405"))
}

// maybeCaptureScreens memotret tiap layar bernama dgn DATA NYATA: simpan state,
// arahkan UI ke layar, render headless (ops terpisah → window tak terganggu),
// potret, lalu pulihkan state. wlive-<screen>-<ts>.png.
func (s *shooter) maybeCaptureScreens(ui *gioui.UI, e gioapp.FrameEvent) {
	if s.dir == "" || len(s.screens) == 0 || e.Size.X <= 0 || e.Size.Y <= 0 || time.Since(s.last) < s.every {
		return
	}
	s.last = time.Now()
	if !s.ensureHW(e.Size) {
		return
	}
	savedView, savedOverlay := ui.View(), ui.Overlay()
	ts := s.last.UTC().Format("20060102-150405")
	var ops op.Ops
	for _, scr := range s.screens {
		applyScreen(ui, scr)
		ops.Reset()
		gtx := gioapp.NewContext(&ops, e)
		ui.Layout(gtx) // bangun layar bernama dgn data nyata
		s.snap(&ops, e.Size, ts+"-"+scr)
	}
	ui.SetView(savedView) // pulihkan tampilan user
	ui.SetOverlay(savedOverlay)
}

// applyScreen mengarahkan UI ke layar bernama (mirror cmd/gio-shot). View vs overlay.
func applyScreen(ui *gioui.UI, name string) {
	switch name {
	case "chats", "calls", "contacts", "status", "channels", "settings":
		ui.SetView(name)
		ui.SetOverlay("")
	case "info", "forward", "picker", "attach", "reaction", "msgctx", "chatctx", "lightbox", "msginfo":
		ui.SetView("chats")
		ui.SetOverlay(name)
	default:
		ui.SetOverlay("")
	}
}
