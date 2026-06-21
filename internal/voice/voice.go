// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// voice.go — putar voice note WhatsApp (ogg-opus) di Gio TANPA libopusfile:
//   ogg pure-Go (parse halaman → paket) + libopus via cgo (decode paket → PCM)
//   + ebitengine/oto (PCM → speaker). Hindari hraban/opus yg butuh opusfile.
//
// cgo butuh: libopus (pkg-config opus). Sudah ada di sistem; tak perlu opusfile.
package voice

/*
#cgo pkg-config: opus
#include <opus/opus.h>
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/oto/v3"
)

const opusRate = 48000 // opus selalu decode @48kHz

// --- decoder libopus (cgo) ---
type opusDec struct{ d *C.OpusDecoder }

func newOpusDec(ch int) (*opusDec, error) {
	var cerr C.int
	d := C.opus_decoder_create(C.opus_int32(opusRate), C.int(ch), &cerr)
	if cerr != 0 || d == nil {
		return nil, fmt.Errorf("opus_decoder_create: %d", int(cerr))
	}
	return &opusDec{d}, nil
}
func (o *opusDec) close() {
	if o.d != nil {
		C.opus_decoder_destroy(o.d)
		o.d = nil
	}
}

// decode satu paket opus → sampel PCM int16 (interleaved).
func (o *opusDec) decode(pkt []byte, ch int) []int16 {
	const maxFrame = 5760 // 120ms @48k
	pcm := make([]int16, maxFrame*ch)
	var pp *C.uchar
	if len(pkt) > 0 {
		pp = (*C.uchar)(unsafe.Pointer(&pkt[0]))
	}
	n := C.opus_decode(o.d, pp, C.opus_int32(len(pkt)),
		(*C.opus_int16)(unsafe.Pointer(&pcm[0])), C.int(maxFrame), 0)
	if n <= 0 {
		return nil
	}
	return pcm[:int(n)*ch]
}

// --- parser Ogg pure-Go (ekstrak paket dari halaman OggS) ---
func oggPackets(data []byte) [][]byte {
	var packets [][]byte
	var cur []byte
	i := 0
	for i+27 <= len(data) {
		if string(data[i:i+4]) != "OggS" {
			break
		}
		nseg := int(data[i+26])
		if i+27+nseg > len(data) {
			break
		}
		segTable := data[i+27 : i+27+nseg]
		body := data[i+27+nseg:]
		off := 0
		for _, s := range segTable {
			if off+int(s) > len(body) {
				return packets
			}
			cur = append(cur, body[off:off+int(s)]...)
			off += int(s)
			if s < 255 { // paket selesai (segmen <255 = akhir)
				packets = append(packets, cur)
				cur = nil
			}
		}
		i += 27 + nseg + off
	}
	return packets
}

// DecodePCM: ogg-opus byte → (PCM bytes int16LE, channels). Murni decode (no I/O).
func DecodePCM(oggData []byte) ([]byte, int, error) {
	pkts := oggPackets(oggData)
	if len(pkts) < 3 {
		return nil, 0, errors.New("voice: tak ada paket audio opus")
	}
	ch := 1
	if len(pkts[0]) >= 10 && string(pkts[0][:8]) == "OpusHead" {
		ch = int(pkts[0][9])
		if ch < 1 || ch > 2 {
			ch = 1
		}
	}
	dec, err := newOpusDec(ch)
	if err != nil {
		return nil, 0, err
	}
	defer dec.close()
	var buf bytes.Buffer
	tmp := make([]byte, 0, 4096)
	for _, p := range pkts[2:] { // lewati OpusHead + OpusTags
		s := dec.decode(p, ch)
		tmp = tmp[:0]
		for _, v := range s {
			tmp = append(tmp, byte(uint16(v)), byte(uint16(v)>>8))
		}
		buf.Write(tmp)
	}
	if buf.Len() == 0 {
		return nil, 0, errors.New("voice: decode kosong")
	}
	return buf.Bytes(), ch, nil
}

// --- pemutar (oto) ---
var (
	ctxOnce sync.Once
	otoCtx  *oto.Context
	ctxErr  error
)

func ensureCtx(ch int) (*oto.Context, error) {
	ctxOnce.Do(func() {
		var ready chan struct{}
		otoCtx, ready, ctxErr = oto.NewContext(&oto.NewContextOptions{
			SampleRate:   opusRate,
			ChannelCount: ch,
			Format:       oto.FormatSignedInt16LE,
		})
		if ctxErr == nil {
			<-ready
		}
	})
	return otoCtx, ctxErr
}

// Play: ogg-opus byte → speaker (async; kembalikan stop-func). chans diasumsikan
// sama utk seluruh sesi (mono umum di WA).
func Play(oggData []byte) (func(), error) {
	pcm, ch, err := DecodePCM(oggData)
	if err != nil {
		return nil, err
	}
	c, err := ensureCtx(ch)
	if err != nil {
		return nil, err
	}
	p := c.NewPlayer(bytes.NewReader(pcm))
	p.Play()
	stop := func() { _ = p.Close() }
	// auto-close saat selesai (durasi = sampel/rate).
	dur := time.Duration(len(pcm)/2/ch) * time.Second / opusRate
	go func() { time.Sleep(dur + 200*time.Millisecond); _ = p.Close() }()
	return stop, nil
}

var _ io.Reader = (*bytes.Reader)(nil)
var _ = binary.LittleEndian
