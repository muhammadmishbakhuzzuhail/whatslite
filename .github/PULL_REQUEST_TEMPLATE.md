## Ringkasan
Apa yang diubah dan kenapa.

## Jenis
- [ ] fix (perbaikan bug)
- [ ] feat (fitur baru)
- [ ] perf / refactor
- [ ] docs / chore

## Checklist (wajib — sama dengan CI)
- [ ] `go vet ./...` bersih
- [ ] `go test ./...` lulus
- [ ] `npm --prefix frontend run lint:css` 0 error
- [ ] `npm --prefix frontend run build` sukses, **0 warning unused-CSS**
- [ ] `wails build -tags "webkit2_41 netgo"` sukses
- [ ] Tidak menambah dependensi berat tanpa alasan kuat (lihat [filosofi lean](../CONTRIBUTING.md#filosofi-tetap-lean))
- [ ] Tidak ada rahasia / `*.db` / binary ter-commit
- [ ] Kerja DB berat di event handler di-offload ke `a.bg()` (handler whatsmeow sinkron)

## Catatan tambahan
Screenshot/GIF jika ada perubahan UI.
