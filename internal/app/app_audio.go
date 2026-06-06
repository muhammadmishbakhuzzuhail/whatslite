package app

// app_audio.go — transcode voice note ke OGG/Opus (format PTT WhatsApp) via
// ffmpeg. WebKitGTK sering merekam webm/opus; ponsel WhatsApp tak memutar PTT
// webm. Best-effort: kalau ffmpeg tak ada / gagal, pemanggil pakai data asli.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// transcribeAudio mengubah audio voice → teks (STT lokal, best-effort).
// Butuh whisper.cpp (`whisper-cli`/`main`) + model via env WALITE_WHISPER_MODEL,
// ATAU openai-whisper (`whisper`). Tanpa itu → ("", false). ffmpeg dipakai utk
// konversi ke wav 16k mono. Privasi: semua diproses LOKAL (tak ada cloud).
func transcribeAudio(ctx context.Context, data []byte) (string, bool) {
	if len(data) == 0 {
		return "", false
	}
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", false
	}
	tmp, err := os.MkdirTemp("", "walite-stt")
	if err != nil {
		return "", false
	}
	defer os.RemoveAll(tmp)
	wav := filepath.Join(tmp, "a.wav")
	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	conv := exec.CommandContext(cctx, ff, "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-ar", "16000", "-ac", "1", "-f", "wav", wav)
	conv.Stdin = bytes.NewReader(data)
	if err := conv.Run(); err != nil {
		return "", false
	}
	model := os.Getenv("WALITE_WHISPER_MODEL")
	// whisper.cpp: whisper-cli / main -m model -f wav -otxt -of out -nt
	for _, bin := range []string{"whisper-cli", "whisper-cpp", "main"} {
		if p, e := exec.LookPath(bin); e == nil && model != "" {
			of := filepath.Join(tmp, "out")
			c := exec.CommandContext(cctx, p, "-m", model, "-f", wav, "-otxt", "-nt", "-of", of)
			if c.Run() == nil {
				if b, e := os.ReadFile(of + ".txt"); e == nil {
					return strings.TrimSpace(string(b)), true
				}
			}
		}
	}
	// openai-whisper: whisper a.wav --output_format txt --output_dir tmp
	if p, e := exec.LookPath("whisper"); e == nil {
		c := exec.CommandContext(cctx, p, wav, "--output_format", "txt", "--output_dir", tmp, "--model", "base")
		if c.Run() == nil {
			if b, e := os.ReadFile(filepath.Join(tmp, "a.txt")); e == nil {
				return strings.TrimSpace(string(b)), true
			}
		}
	}
	return "", false
}

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
		// -loop 0 = animasi berputar terus (default libwebp kadang loop sekali →
		// stiker animasi berhenti). -an buang audio (mp4 stiker animasi).
		"-an", "-loop", "0", "-c:v", "libwebp", "-pix_fmt", "yuva420p", "-q:v", "75", "-f", "webp", "pipe:1",
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
