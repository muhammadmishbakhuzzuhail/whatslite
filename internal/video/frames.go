// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// frames.go — decode video → aliran frame RGBA via ffmpeg (pure os/exec, tanpa
// cgo) untuk PUTAR INLINE (mis. viewer status), bukan window mpv terpisah. Audio
// disetel terpisah (PlayAudioOnly, libmpv vid=no → tanpa window). Streaming
// producer→channel; konsumen (UI) ambil frame menurut waktu main.
package video

import (
	"bufio"
	"context"
	"encoding/json"
	"image"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// FrameStream — aliran frame RGBA satu video. Frame(elapsed) memajukan ke frame
// yang sesuai waktu main; ended=true saat habis.
type FrameStream struct {
	W, H int
	fps  int
	dur  time.Duration

	ch     chan *image.RGBA
	cur    *image.RGBA
	curIdx int
	ended  bool

	cancel context.CancelFunc
	tmp    string
}

// OpenFrames — decode `data` (ext mis. ".mp4") → FrameStream lebar maks maxW, fps
// tetap. nil+err bila ffmpeg/ffprobe absen atau gagal probe.
func OpenFrames(data []byte, ext string, maxW int) (*FrameStream, error) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "wlvid-*"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, err
	}
	tmp.Close()

	w, h, dur := probe(tmp.Name())
	if w <= 0 || h <= 0 {
		w, h = 720, 1280 // fallback potret
	}
	// skala ke maxW (genap), tinggi proporsional (genap).
	ow := w
	if ow > maxW {
		ow = maxW
	}
	ow &^= 1
	oh := h * ow / w
	oh &^= 1
	if oh <= 0 {
		oh = 2
	}
	const fps = 24

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, ff, "-hide_banner", "-loglevel", "error",
		"-i", tmp.Name(),
		"-vf", "scale="+itoa(ow)+":"+itoa(oh)+",fps="+itoa(fps),
		"-f", "rawvideo", "-pix_fmt", "rgba", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		os.Remove(tmp.Name())
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		os.Remove(tmp.Name())
		return nil, err
	}
	fs := &FrameStream{W: ow, H: oh, fps: fps, dur: dur, ch: make(chan *image.RGBA, 48), cancel: cancel, tmp: tmp.Name()}
	go fs.decode(bufio.NewReaderSize(stdout, ow*oh*4), cmd)
	return fs, nil
}

// decode membaca frame raw RGBA terus-menerus → channel (blocking saat penuh →
// jeda alami bila UI pause). Tutup channel saat EOF.
func (fs *FrameStream) decode(r *bufio.Reader, cmd *exec.Cmd) {
	defer close(fs.ch)
	defer cmd.Wait()
	defer os.Remove(fs.tmp)
	sz := fs.W * fs.H * 4
	for {
		buf := make([]byte, sz)
		n := 0
		for n < sz { // baca penuh satu frame
			m, err := r.Read(buf[n:])
			if m > 0 {
				n += m
			}
			if err != nil {
				return // EOF / dibatalkan
			}
		}
		img := &image.RGBA{Pix: buf, Stride: fs.W * 4, Rect: image.Rect(0, 0, fs.W, fs.H)}
		select {
		case fs.ch <- img:
		case <-time.After(30 * time.Second): // konsumen hilang → stop
			return
		}
	}
}

// Frame — frame untuk waktu-main `elapsed`. Memajukan curIdx ke target fps; bila
// channel belum siap, kembalikan frame terakhir. ended=true saat aliran habis.
func (fs *FrameStream) Frame(elapsed time.Duration) (*image.RGBA, bool) {
	want := int(elapsed.Seconds() * float64(fs.fps))
	for fs.curIdx <= want && !fs.ended {
		select {
		case fr, ok := <-fs.ch:
			if !ok {
				fs.ended = true
			} else {
				fs.cur = fr
				fs.curIdx++
			}
		default:
			return fs.cur, fs.ended // belum ada frame baru
		}
	}
	return fs.cur, fs.ended
}

func (fs *FrameStream) Duration() time.Duration { return fs.dur }

func (fs *FrameStream) Close() {
	if fs.cancel != nil {
		fs.cancel()
	}
}

// probe — width,height,duration via ffprobe (JSON). 0/0/0 bila absen/gagal.
func probe(path string) (w, h int, dur time.Duration) {
	fp, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, 0, 0
	}
	out, err := exec.Command(fp, "-v", "quiet", "-print_format", "json",
		"-show_streams", "-show_format", "-select_streams", "v:0", path).Output()
	if err != nil {
		return 0, 0, 0
	}
	var p struct {
		Streams []struct {
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(out, &p) != nil || len(p.Streams) == 0 {
		return 0, 0, 0
	}
	w, h = p.Streams[0].Width, p.Streams[0].Height
	ds := p.Streams[0].Duration
	if ds == "" {
		ds = p.Format.Duration
	}
	if f, e := strconv.ParseFloat(ds, 64); e == nil && f > 0 {
		dur = time.Duration(f * float64(time.Second))
	}
	return w, h, dur
}

func itoa(n int) string { return strconv.Itoa(n) }
