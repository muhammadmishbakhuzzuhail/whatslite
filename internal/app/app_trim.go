// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_trim.go — potong (trim) durasi media (video/audio) via ffmpeg stream-copy
// (-ss/-t -c copy): cepat, tanpa re-encode (presisi ke keyframe terdekat). Dipakai
// pratinjau kirim (video trim) + nanti voice trim. ffmpeg absen → "".

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TrimMedia memotong media data-URI ke [startSec, endSec]. endSec<=startSec →
// dari startSec sampai akhir. start=0 & end=0 → tak ada potong (kembalikan asli).
// Kembalikan data-URI baru, atau "" bila gagal/ffmpeg absen.
func TrimMediaURI(dataURI string, startSec, endSec float64) string {
	if startSec <= 0 && endSec <= 0 {
		return dataURI // tak ada perubahan
	}
	mime, data, err := decodeDataURI(dataURI)
	if err != nil || len(data) == 0 {
		return ""
	}
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ""
	}
	ext := ".mp4"
	switch {
	case strings.Contains(mime, "webm"):
		ext = ".webm"
	case strings.Contains(mime, "ogg"), strings.Contains(mime, "opus"):
		ext = ".ogg"
	case strings.Contains(mime, "mp4a"), strings.Contains(mime, "m4a"), strings.Contains(mime, "aac"):
		ext = ".m4a"
	case strings.Contains(mime, "mpeg"), strings.Contains(mime, "mp3"):
		ext = ".mp3"
	case strings.Contains(mime, "quicktime"), strings.Contains(mime, "mov"):
		ext = ".mov"
	}
	inF, err := os.CreateTemp("", "trimin-*"+ext)
	if err != nil {
		return ""
	}
	defer os.Remove(inF.Name())
	_, _ = inF.Write(data)
	inF.Close()
	outF, err := os.CreateTemp("", "trimout-*"+ext)
	if err != nil {
		return ""
	}
	outPath := outF.Name()
	outF.Close()
	defer os.Remove(outPath)

	cctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if startSec > 0 {
		args = append(args, "-ss", strconv.FormatFloat(startSec, 'f', 3, 64))
	}
	args = append(args, "-i", inF.Name())
	if endSec > startSec {
		args = append(args, "-t", strconv.FormatFloat(endSec-startSec, 'f', 3, 64))
	}
	args = append(args, "-c", "copy", "-movflags", "+faststart", outPath)
	if err := exec.CommandContext(cctx, bin, args...).Run(); err != nil {
		return ""
	}
	out, err := os.ReadFile(outPath)
	if err != nil || len(out) == 0 {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(out)
}

// TrimMedia (metode App) — bungkus TrimMediaURI agar bisa dipanggil UI lewat core.
func (a *App) TrimMedia(dataURI string, startSec, endSec float64) string {
	return TrimMediaURI(dataURI, startSec, endSec)
}
