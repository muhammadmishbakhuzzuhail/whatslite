// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// retry.go — retry + exponential backoff utk operasi jaringan transien
// (upload/download media, fetch foto CDN). whatsmeow menangani reconnect socket,
// tapi operasi sekali-jalan ini gagal keras tanpa percobaan ulang.

import (
	"context"
	"time"
)

// retry menjalankan fn hingga sukses atau attempts habis, dgn backoff
// eksponensial (400ms, 800ms, 1.6s, …). Berhenti awal bila ctx dibatalkan.
// Mengembalikan error percobaan terakhir.
func retry(ctx context.Context, attempts int, fn func() error) error {
	if attempts < 1 {
		attempts = 1
	}
	delay := 400 * time.Millisecond
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
	}
	return err
}
