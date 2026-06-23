// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_camera.go — ambil foto dari webcam tanpa dependency baru: ffmpeg `-f v4l2`
// menangkap satu frame JPEG (lewati ~4 frame warmup yg sering hijau/gelap) →
// data-URI utk masuk pratinjau media (caption + sekali-lihat) lalu dikirim.

import (
	"bytes"
	"context"
	"encoding/base64"
	"os/exec"
	"time"
)

// CapturePhoto — tangkap satu foto webcam → data-URI JPEG. "" bila ffmpeg absen /
// kamera gagal. Coba /dev/video0 lalu /dev/video1.
func (a *App) CapturePhoto() string {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		a.emit("wa:error", "ffmpeg tak ditemukan (perlu utk kamera)")
		return ""
	}
	for _, dev := range []string{"/dev/video0", "/dev/video1"} {
		cctx, cancel := context.WithTimeout(a.ctx, 8*time.Second)
		cmd := exec.CommandContext(cctx, bin, "-hide_banner", "-loglevel", "error",
			"-f", "v4l2", "-i", dev,
			// lewati 4 frame warmup (sering hijau/gelap) → ambil frame ke-5.
			"-vf", `select=gte(n\,4)`, "-frames:v", "1", "-vsync", "0",
			"-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1")
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		err := cmd.Run()
		cancel()
		if err == nil && out.Len() > 0 {
			return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
		}
	}
	a.emit("wa:error", "tak bisa membuka kamera")
	return ""
}
