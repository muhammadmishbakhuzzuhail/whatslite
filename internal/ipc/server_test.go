// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package ipc

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

// TestBroadcast membuktikan klien FE (non-Wails) menerima event lewat UDS
// sebagai NDJSON {"@type":...,"payload":...} — inti jembatan dual-emit.
func TestBroadcast(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "bridge.sock")
	srv, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer srv.Close()

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	// Beri waktu acceptLoop mendaftarkan klien sebelum broadcast.
	time.Sleep(50 * time.Millisecond)
	srv.Broadcast("wa:message", "123@s.whatsapp.net")

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	var f struct {
		Type    string `json:"@type"`
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(line, &f); err != nil {
		t.Fatalf("Unmarshal %q: %v", line, err)
	}
	if f.Type != "wa:message" || f.Payload != "123@s.whatsapp.net" {
		t.Fatalf("got type=%q payload=%q", f.Type, f.Payload)
	}
}

// TestRequestResponse membuktikan arah FE→engine: klien kirim request
// {@type,@extra,args}, handler dipanggil dgn method+args, klien terima Response
// dgn @extra yang sama (korelasi) → fondasi RPC ala TDLib.
func TestRequestResponse(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "bridge.sock")
	srv, err := Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer srv.Close()

	var gotMethod string
	var gotArgN int
	srv.SetHandler(func(method string, args []json.RawMessage) (any, error) {
		gotMethod = method
		gotArgN = len(args)
		return map[string]string{"ok": method}, nil
	})

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	if _, err := conn.Write([]byte(`{"@type":"GetChats","@extra":"7","args":["x",true]}` + "\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	var resp struct {
		Type   string            `json:"@type"`
		Extra  string            `json:"@extra"`
		Result map[string]string `json:"result"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("Unmarshal %q: %v", line, err)
	}
	if resp.Type != "Response" || resp.Extra != "7" || resp.Result["ok"] != "GetChats" {
		t.Fatalf("bad response: %q", line)
	}
	if gotMethod != "GetChats" || gotArgN != 2 {
		t.Fatalf("handler got method=%q args=%d", gotMethod, gotArgN)
	}
}
