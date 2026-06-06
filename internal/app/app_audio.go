package app

// app_audio.go — transcode voice note ke OGG/Opus (format PTT WhatsApp) via
// ffmpeg. WebKitGTK sering merekam webm/opus; ponsel WhatsApp tak memutar PTT
// webm. Best-effort: kalau ffmpeg tak ada / gagal, pemanggil pakai data asli.

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// transcodeToWebpSticker → WebP 512² transparan (format stiker WhatsApp) dari
// PNG/gif/dll via ffmpeg. (nil,false) bila ffmpeg absen/gagal.
func transcodeToWebpSticker(ctx context.Context, data []byte) ([]byte, bool) {
	if len(data) == 0 {
		return nil, false
	}
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, false
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin,
		"-hide_banner", "-loglevel", "error", "-i", "pipe:0",
		"-vf", "scale=512:512:force_original_aspect_ratio=decrease,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=0x00000000",
		"-c:v", "libwebp", "-pix_fmt", "yuva420p", "-q:v", "75", "-f", "webp", "pipe:1",
	)
	cmd.Stdin = bytes.NewReader(data)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil || out.Len() == 0 {
		return nil, false
	}
	return out.Bytes(), true
}

// mmss memformat detik → "m:ss" (durasi voice note di bubble lokal).
func mmss(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// transcodeToOggOpus mengubah byte audio (mis. webm/opus) → ogg/opus mono 48k.
// Mengembalikan (oggBytes, true) bila sukses; (nil, false) bila ffmpeg absen/gagal.
func transcodeToOggOpus(ctx context.Context, data []byte) ([]byte, bool) {
	if len(data) == 0 {
		return nil, false
	}
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, false
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// pipe:0 → stdin, pipe:1 → stdout. Re-encode opus mono 48k ~32kbps (PTT).
	cmd := exec.CommandContext(cctx, bin,
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-vn", "-ac", "1", "-ar", "48000", "-c:a", "libopus", "-b:a", "32k",
		"-f", "ogg", "pipe:1",
	)
	cmd.Stdin = bytes.NewReader(data)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil || out.Len() == 0 {
		return nil, false
	}
	return out.Bytes(), true
}
