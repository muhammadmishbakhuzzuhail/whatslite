// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// webp.go — utak-atik loop-count animated WebP TANPA re-encode.
//
// Stiker animasi WhatsApp = WebP animasi. Sumber (KLIPY dll) kadang menyimpan
// loop-count = 1 → stiker main sekali lalu berhenti. ffmpeg tak bisa bantu:
// decoder webp ffmpeg hanya baca frame pertama (re-mux malah membekukan).
// Solusi: tulis ulang 2 byte loop-count di chunk ANIM jadi 0 (= tak terbatas).
//
// Format kontainer WebP (RIFF):
//   [0:4]  "RIFF"  [4:8] ukuran-file(LE)  [8:12] "WEBP"
//   lalu rentetan chunk: [FourCC 4][size LE 4][payload (dipad ke genap)]
// Chunk ANIM: payload = bgcolor(4) + loop_count(2, LE). loop_count=0 = loop tanpa henti.

import "encoding/binary"

// webpLoopForever menyetel loop-count animated WebP → 0 (tak terbatas).
// Mengembalikan slice yang dimodifikasi in-place. Bila bukan WebP / bukan
// animasi (tak ada chunk ANIM) / parse gagal → kembalikan data apa adanya.
func webpLoopForever(data []byte) []byte {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return data
	}
	off := 12
	for off+8 <= len(data) {
		fourcc := string(data[off : off+4])
		size := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		payload := off + 8
		if payload+size > len(data) {
			break
		}
		if fourcc == "ANIM" && size >= 6 {
			// loop_count = payload[4:6] (LE). 0 = tak terbatas.
			binary.LittleEndian.PutUint16(data[payload+4:payload+6], 0)
			return data
		}
		// chunk size ganjil → ada 1 byte pad.
		off = payload + size + (size & 1)
	}
	return data
}
