// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_voicerec.go — rekam voice note (PTT) tanpa dependency baru: tangkap mic via
// ffmpeg (PulseAudio/PipeWire `pulse`, fallback `alsa`) langsung ke OGG/Opus mono
// 48k 32kbps (format PTT WhatsApp), lalu kirim via SendMedia kind "voice".
//   StartVoiceRecord()        → mulai rekam
//   StopVoiceRecordAndSend()  → finalisasi + kirim
//   CancelVoiceRecord()       → buang
// ffmpeg difinalisasi via SIGINT agar trailer OGG tertulis (file valid).

import (
	"encoding/base64"
	"os"
	"os/exec"
	"sync"
	"time"
)

type voiceRec struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	tmp   string
	start time.Time
}

// VoiceRecording — true bila sedang merekam (UI: tampil bar rekam).
func (a *App) VoiceRecording() bool {
	a.vrec.mu.Lock()
	defer a.vrec.mu.Unlock()
	return a.vrec.cmd != nil
}

// VoiceRecordSeconds — durasi rekaman berjalan (detik) utk timer UI.
func (a *App) VoiceRecordSeconds() int {
	a.vrec.mu.Lock()
	defer a.vrec.mu.Unlock()
	if a.vrec.cmd == nil {
		return 0
	}
	return int(time.Since(a.vrec.start).Seconds())
}

// StartVoiceRecord — mulai tangkap mic ke OGG/Opus. false bila ffmpeg absen / mic
// gagal dibuka. Coba `pulse` dulu (PipeWire-pulse umum di GNOME), lalu `alsa`.
func (a *App) StartVoiceRecord() bool {
	a.vrec.mu.Lock()
	defer a.vrec.mu.Unlock()
	if a.vrec.cmd != nil {
		return true // sudah merekam
	}
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		a.emit("wa:error", "ffmpeg tak ditemukan (perlu utk rekam suara)")
		return false
	}
	f, err := os.CreateTemp("", "walite-ptt-*.ogg")
	if err != nil {
		return false
	}
	tmp := f.Name()
	f.Close()

	for _, dev := range []struct{ fmt, src string }{{"pulse", "default"}, {"alsa", "default"}} {
		cmd := exec.Command(bin, "-hide_banner", "-loglevel", "error",
			"-f", dev.fmt, "-i", dev.src,
			"-ac", "1", "-ar", "48000", "-c:a", "libopus", "-b:a", "32k", "-f", "ogg", "-y", tmp)
		if err := cmd.Start(); err != nil {
			continue
		}
		// ffmpeg bisa keluar segera bila device gagal → cek sebentar.
		time.Sleep(150 * time.Millisecond)
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			continue // device ini gagal, coba berikutnya
		}
		a.vrec.cmd, a.vrec.tmp, a.vrec.start = cmd, tmp, time.Now()
		return true
	}
	_ = os.Remove(tmp)
	a.emit("wa:error", "tak bisa membuka mikrofon")
	return false
}

// stop — hentikan ffmpeg dgn SIGINT (tulis trailer OGG), tunggu, kembalikan path +
// durasi. cmd/tmp di-reset. Pemanggil memegang/melepas mu sendiri.
func (a *App) stopRec() (tmp string, secs int) {
	a.vrec.mu.Lock()
	cmd, tmp, start := a.vrec.cmd, a.vrec.tmp, a.vrec.start
	a.vrec.cmd, a.vrec.tmp = nil, ""
	a.vrec.mu.Unlock()
	if cmd == nil {
		return "", 0
	}
	_ = cmd.Process.Signal(os.Interrupt) // ffmpeg finalisasi OGG saat SIGINT
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
	return tmp, int(time.Since(start).Seconds())
}

// StopVoiceRecordAndSend — finalisasi rekaman + kirim sbg voice note ke chat.
func (a *App) StopVoiceRecordAndSend(chat string) {
	tmp, secs := a.stopRec()
	if tmp == "" {
		return
	}
	defer os.Remove(tmp)
	data, err := os.ReadFile(tmp)
	if err != nil || len(data) == 0 {
		return
	}
	if secs < 1 {
		secs = 1
	}
	uri := "data:audio/ogg;base64," + base64.StdEncoding.EncodeToString(data)
	a.SendMedia(chat, "voice", "", "", uri, false, secs)
}

// CancelVoiceRecord — hentikan + buang rekaman (tak dikirim).
func (a *App) CancelVoiceRecord() {
	tmp, _ := a.stopRec()
	if tmp != "" {
		_ = os.Remove(tmp)
	}
}
