// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

// Package engine adalah inti WhatsLite: pembungkus whatsmeow + penyimpanan.
//
// Engine sengaja dibuat AGNOSTIK terhadap frontend (tidak tahu soal GUI/TUI)
// supaya bisa dipakai oleh GUI Wails, TUI, maupun mode daemon headless nanti.
// Tipe-tipe whatsmeow tidak bocor keluar paket ini.
//
// File-file paket (per domain, untuk memudahkan dokumentasi & perbaikan):
//   - engine.go    : siklus hidup (New/Start/Stop/Logout) + util data dir
//   - names.go     : resolusi nama chat/kontak (@lid bridge), grup, app-state
//   - messages.go  : pesan masuk/keluar + history sync + klasifikasi konten
//   - media.go     : foto profil + unduh media
//   - presence.go  : online/typing/receipt + event koneksi
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	// Driver SQLite pure-Go (tanpa CGo) -> binary statis & portabel lintas distro.
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Engine memegang koneksi WhatsApp dan penyimpanan lokal.
type Engine struct {
	Client    *whatsmeow.Client
	container *sqlstore.Container

	mu         sync.Mutex
	groupNames map[string]string // cache subjek grup (jid -> nama)
}

// QREvent adalah event pairing yang disederhanakan (tanpa tipe whatsmeow).
type QREvent struct {
	Event string // "code", "success", "timeout", "error", dll.
	Code  string // berisi data QR mentah saat Event == "code"
	Err   error  // berisi error saat Event == "error"
}

// New membuat Engine dengan store SQLite di dbPath.
//
// Catatan SQLite: driver modernc terdaftar sebagai "sqlite", sedangkan whatsmeow
// memakai string dialek "sqlite3" untuk sintaks query. Maka kita buka *sql.DB
// sendiri dengan driver "sqlite", lalu beri tahu whatsmeow dialeknya "sqlite3".
func New(ctx context.Context, dbPath string, debug bool) (*Engine, error) {
	level := "INFO"
	if debug {
		level = "DEBUG"
	}
	dbLog := waLog.Stdout("DB", level, true)

	// foreign_keys + WAL + busy_timeout = aman & ringan untuk akses lokal.
	// synchronous(NORMAL) aman dgn WAL + jauh lebih cepat saat history-sync
	// menulis ribuan record; temp di RAM; mmap 256MB.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"+
			"&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		dbPath,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite hanya satu penulis; serialkan koneksi untuk hindari "database is locked".
	db.SetMaxOpenConns(1)

	container := sqlstore.NewWithDB(db, "sqlite3", dbLog)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("upgrade db: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", level, true)
	client := whatsmeow.NewClient(device, clientLog)

	// Ketahanan & korektness (riset whatsmeow). EnableAutoReconnect &
	// AutoTrustIdentity sudah default true.
	// - rerequest pesan tak-terdekripsi dari HP → kurangi spam "Error decrypting"
	//   (pesan gagal diisi ulang saat HP online; bkn sekadar hilang).
	client.AutomaticMessageRerequestFromPhone = true
	// - buang duplikat saat backlog offline dikirim ulang pasca-reconnect
	//   (cegah pesan/efek/notif ganda). Buffer dibersihkan tiap 12 jam internal.
	client.EnableDecryptedEventBuffer = true
	// - simpan pesan terkirim ke DB → retry & dekripsi vote-poll tahan restart
	//   (default cuma LRU 256 di memori, hilang saat restart).
	client.UseRetryMessageStore = true

	return &Engine{Client: client, container: container, groupNames: map[string]string{}}, nil
}

// maxGroupNameCache membatasi cache nama grup agar tak tumbuh tanpa batas pada
// akun dgn sangat banyak grup. Saat penuh, buang satu entri (akan di-resolve
// ulang bila diperlukan lagi).
const maxGroupNameCache = 5000

// cacheGroupName menyimpan nama grup ke cache dgn batas ukuran (thread-safe).
func (e *Engine) cacheGroupName(jid, name string) {
	if name == "" {
		return
	}
	e.mu.Lock()
	if len(e.groupNames) >= maxGroupNameCache {
		for k := range e.groupNames {
			delete(e.groupNames, k)
			break
		}
	}
	e.groupNames[jid] = name
	e.mu.Unlock()
}

// NeedsLogin melaporkan apakah pairing (scan QR) masih diperlukan.
func (e *Engine) NeedsLogin() bool {
	return e.Client.Store.ID == nil
}

// Start menghubungkan ke WhatsApp.
//
// Jika perlu login, mengembalikan channel QREvent (range sampai tertutup).
// Jika sudah login, channel-nya nil.
func (e *Engine) Start(ctx context.Context) (<-chan QREvent, error) {
	if !e.NeedsLogin() {
		if err := e.Client.Connect(); err != nil {
			return nil, fmt.Errorf("connect: %w", err)
		}
		return nil, nil
	}

	// GetQRChannel HARUS dipanggil sebelum Connect.
	raw, err := e.Client.GetQRChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("qr channel: %w", err)
	}
	if err := e.Client.Connect(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	out := make(chan QREvent)
	go func() {
		defer close(out)
		for evt := range raw {
			out <- QREvent{Event: evt.Event, Code: evt.Code, Err: evt.Error}
		}
	}()
	return out, nil
}

// SetProxy menyetel proxy HTTP/SOCKS (mis. "socks5://127.0.0.1:9050"). Harus
// dipanggil SEBELUM Connect. "" = tanpa proxy.
func (e *Engine) SetProxy(addr string) error {
	if addr == "" {
		return nil
	}
	return e.Client.SetProxyAddress(addr)
}

// PairPhone meminta kode tautan 8-karakter (alternatif QR). Client harus sudah
// Connect (alur QR sudah memanggilnya) & belum ter-pair. Kode diketik di HP:
// Tautkan perangkat → Tautkan dengan nomor telepon. Sukses → event login biasa.
func (e *Engine) PairPhone(ctx context.Context, phone string) (string, error) {
	// Display name WAJIB format "Browser (OS)" — server tolak 400 kalau bukan
	// browser/OS umum (mis. "WhatsLite" gagal). Pakai "Chrome (Linux)".
	return e.Client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
}

// Logout memutus tautan perangkat (unpair). Setelah ini NeedsLogin() == true,
// jadi untuk masuk lagi perlu Start() ulang (QR baru) — berguna untuk
// "keluar akun" maupun ganti akun.
func (e *Engine) Logout(ctx context.Context) error {
	return e.Client.Logout(ctx)
}

// SelfJID mengembalikan JID akun yang sedang login (atau string kosong).
func (e *Engine) SelfJID() string {
	if e.Client.Store.ID == nil {
		return ""
	}
	return e.Client.Store.ID.String()
}

// Stop memutus koneksi dengan rapi.
func (e *Engine) Stop() {
	e.Client.Disconnect()
}

// DefaultDataDir mengembalikan direktori data (XDG) dan memastikan ia ada.
// Mengikuti $XDG_DATA_HOME, fallback ke ~/.local/share.
func DefaultDataDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "whatslite")
	// Migrasi rebrand (WhatsApp Lite → WhatsLite): pindahkan data lama sekali,
	// HANYA bila folder baru belum ada & folder lama ada. Non-destruktif: bila
	// gagal, data lama tetap utuh & app mulai bersih (perlu pairing ulang).
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		old := filepath.Join(base, "whatsapp-lite")
		if fi, e := os.Stat(old); e == nil && fi.IsDir() {
			_ = os.Rename(old, dir) // memindah app.db, media/, sesi, kunci API sekaligus
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// Ganti nama file DB sesi whatsmeow: whatsapp-lite.db → whatslite.db (+ WAL/SHM).
	newDB := filepath.Join(dir, "whatslite.db")
	oldDB := filepath.Join(dir, "whatsapp-lite.db")
	if _, err := os.Stat(newDB); os.IsNotExist(err) {
		if _, e := os.Stat(oldDB); e == nil {
			_ = os.Rename(oldDB, newDB)
			_ = os.Rename(oldDB+"-wal", newDB+"-wal")
			_ = os.Rename(oldDB+"-shm", newDB+"-shm")
		}
	}
	return dir, nil
}
