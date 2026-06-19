// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package app

// app_ipc.go — dispatcher request IPC (FE-baru Qt → engine). Memetakan frame
// {"@type":"GetChats","@extra":"7","args":[...]} ke method publik App lewat
// REFLEKSI — sama seperti yang dilakukan Wails ke JS, jadi ke-~110 method
// otomatis terjangkau tanpa switch manual yang harus disinkronkan tangan.
//
// Keamanan: socket UDS 0600 (pengguna lokal sama, setara kepercayaan Wails).
// Method siklus-hidup di-denylist agar FE tak bisa memicunya.

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/ipc"
)

// ipcDenylist = method yang TAK boleh dipanggil dari FE (siklus hidup/internal).
var ipcDenylist = map[string]bool{
	"Startup": true, "DomReady": true, "Shutdown": true, "BeforeClose": true,
	"ServeHTTP": true, "SetVersion": true, "StartupHeadless": true,
	// Method khusus-Wails (kelola window) → FE Qt kelola window sendiri; blokir
	// agar tak memanggil runtime.* dgn ctx non-Wails (panik di headless).
	"Quit": true, "SetUnreadBadge": true,
}

var errIface = reflect.TypeOf((*error)(nil)).Elem()

// dispatchIPC memanggil method App by-name dgn argumen JSON. Mengembalikan nilai
// balik non-error sebagai result + error (bila ada) → server balas Response/Error.
func (a *App) dispatchIPC(method string, rawArgs []json.RawMessage) (any, error) {
	if ipcDenylist[method] {
		return nil, fmt.Errorf("method tak diizinkan: %s", method)
	}
	m := reflect.ValueOf(a).MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("method tak dikenal: %s", method)
	}
	mt := m.Type()
	if mt.IsVariadic() {
		return nil, fmt.Errorf("method variadic tak didukung: %s", method)
	}
	if len(rawArgs) != mt.NumIn() {
		return nil, fmt.Errorf("%s: butuh %d argumen, dapat %d", method, mt.NumIn(), len(rawArgs))
	}

	in := make([]reflect.Value, mt.NumIn())
	for i := 0; i < mt.NumIn(); i++ {
		pv := reflect.New(mt.In(i)) // *T
		if len(rawArgs[i]) > 0 {
			if err := json.Unmarshal(rawArgs[i], pv.Interface()); err != nil {
				return nil, fmt.Errorf("%s arg %d: %w", method, i, err)
			}
		}
		in[i] = pv.Elem()
	}

	out := m.Call(in)
	var result any
	var errOut error
	for _, r := range out {
		if r.Type().Implements(errIface) {
			if !r.IsNil() {
				errOut = r.Interface().(error)
			}
			continue
		}
		result = r.Interface() // nilai balik non-error terakhir = hasil
	}
	return result, errOut
}

// attachIPC memasang dispatcher ke server IPC (dipanggil setelah ipc.Listen).
func (a *App) attachIPC(srv *ipc.Server) {
	srv.SetHandler(a.dispatchIPC)
	a.ipc = srv
}
