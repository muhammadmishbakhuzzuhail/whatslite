// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

// Package ipc adalah jembatan engine↔FE-baru (Qt6/QML) lewat Unix domain
// socket dgn frame NDJSON (satu objek JSON per baris). Modelnya meniru TDLib:
// satu stream dua arah — request (FE→engine, @type=method, @extra=id korelasi)
// + event/response (engine→FE). FASE-1 di sini: STREAM EVENT saja (Broadcast).
// Dispatch ~110 method request menyusul (fase-2, di-codegen dari method App).
//
// Strategi migrasi: engine emit event ke DUA jalur sekaligus (Wails/Svelte DAN
// IPC ini) lewat App.emit → Svelte tetap jalan penuh, Qt baru ikut dengar.
// Rollback = gratis (binary Wails tetap bisa di-build).
package ipc

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"sync"
)

// frame = satu baris NDJSON. Event: {"@type":"wa:message","payload":...}.
type frame struct {
	Type    string `json:"@type"`
	Payload any    `json:"payload,omitempty"`
}

// reqFrame = request FE→engine: @type=nama method, @extra=id korelasi,
// args=array argumen JSON (satu per parameter method).
type reqFrame struct {
	Type  string            `json:"@type"`
	Extra string            `json:"@extra"`
	Args  []json.RawMessage `json:"args"`
}

// RequestFunc memproses satu request → (hasil, error). Dipasang oleh paket app
// (dispatch via refleksi ke method App) agar ipc tetap generic (tanpa import app).
type RequestFunc func(method string, args []json.RawMessage) (any, error)

// Server: stream event (Broadcast) + dispatch request (handler) ke klien FE/UDS.
type Server struct {
	ln      net.Listener
	mu      sync.Mutex
	clients map[net.Conn]*bufio.Writer
	handler RequestFunc
}

// SetHandler memasang dispatcher request (FE→engine). nil = abaikan request.
func (s *Server) SetHandler(h RequestFunc) { s.handler = h }

// Listen membuka UDS di sockPath (hapus sisa lama) lalu menerima koneksi.
func Listen(sockPath string) (*Server, error) {
	_ = os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}
	// Hanya pemilik (FE lokal pengguna sama) yang boleh dial.
	_ = os.Chmod(sockPath, 0o600)
	s := &Server{ln: ln, clients: map[net.Conn]*bufio.Writer{}}
	go s.acceptLoop()
	return s, nil
}

func (s *Server) acceptLoop() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return // listener ditutup
		}
		s.mu.Lock()
		s.clients[c] = bufio.NewWriter(c)
		s.mu.Unlock()
		go s.readLoop(c)
	}
}

// readLoop membaca frame request per-baris (NDJSON) → dispatch via handler →
// balas Response/Error dgn @extra yang sama. Satu goroutine per klien:
// request klien berurutan; klien lain tak terblok.
func (s *Server) readLoop(c net.Conn) {
	sc := bufio.NewScanner(c)
	// Frame request bisa besar (mis. SendMedia data-URI) → naikkan batas baris.
	sc.Buffer(make([]byte, 0, 64*1024), 16<<20)
	for sc.Scan() {
		var req reqFrame
		if json.Unmarshal(sc.Bytes(), &req) != nil || req.Type == "" {
			continue // frame rusak → abaikan (jangan bunuh koneksi)
		}
		if s.handler == nil {
			s.sendTo(c, map[string]any{"@type": "Error", "@extra": req.Extra, "code": "no_handler"})
			continue
		}
		res, err := s.handler(req.Type, req.Args)
		if err != nil {
			s.sendTo(c, map[string]any{"@type": "Error", "@extra": req.Extra, "code": "call_failed", "message": err.Error()})
			continue
		}
		s.sendTo(c, map[string]any{"@type": "Response", "@extra": req.Extra, "result": res})
	}
	s.drop(c)
}

func (s *Server) drop(c net.Conn) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	_ = c.Close()
}

// sendTo menulis satu frame ke SATU klien di bawah lock (serial dgn Broadcast →
// frame tak pernah saling-sela pada koneksi yang sama).
func (s *Server) sendTo(c net.Conn, v any) {
	line, err := json.Marshal(v)
	if err != nil {
		return
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	w := s.clients[c]
	if w == nil {
		return
	}
	if _, err := w.Write(line); err != nil {
		delete(s.clients, c)
		_ = c.Close()
		return
	}
	_ = w.Flush()
}

// Broadcast mengirim satu event ke semua klien (urut per-koneksi terjaga: satu
// writer per conn, ditulis di bawah lock). Klien yang error dibuang.
func (s *Server) Broadcast(typ string, payload any) {
	line, err := json.Marshal(frame{Type: typ, Payload: payload})
	if err != nil {
		return
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	for c, w := range s.clients {
		if _, err := w.Write(line); err != nil {
			delete(s.clients, c)
			_ = c.Close()
			continue
		}
		_ = w.Flush()
	}
}

// Close menghentikan listener (klien terhubung putus sendiri saat proses keluar).
func (s *Server) Close() error {
	if s.ln == nil {
		return nil
	}
	return s.ln.Close()
}
