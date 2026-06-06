package app

// Mode ekspor data nyata (OFFLINE) untuk autopilot/compare:
//
//	whatsapp-lite --export-json <path>
//
// Buka DB store + app.db TANPA menyambung ke WhatsApp (no network → tak kena
// sandbox), reuse logika resolusi nama/urutan yang sama dengan UI (GetChats/
// GetMessages), lalu tulis JSON berbentuk mock.js. Frontend membacanya via
// mode ?data=real → screenshot Chrome file:// menampilkan data nyata + UI nyata
// tanpa perlu display server.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"whatsapp-lite/internal/engine"
	"whatsapp-lite/internal/storage"
)

// runExport menulis snapshot data nyata ke path (JSON). maxChatMsgs = jumlah
// chat teratas yang riwayat pesannya ikut diekspor (sisanya hanya ringkasan).
func RunExport(path string) error {
	ctx := context.Background()

	dataDir, err := engine.DefaultDataDir()
	if err != nil {
		return err
	}
	eng, err := engine.New(ctx, filepath.Join(dataDir, "whatsapp-lite.db"), false)
	if err != nil {
		return err
	}
	defer eng.Stop()
	store, err := storage.New(ctx, filepath.Join(dataDir, "app.db"))
	if err != nil {
		return err
	}
	defer store.Close()

	// ctx Background: GetChats/GetMessages hanya pakai ctx untuk query DB
	// (RecomputeSummaries/ListChats) — runtime.* hanya di jalur error/recover.
	a := &App{ctx: ctx, eng: eng, store: store}

	_ = store.RecomputeSummaries(ctx) // segarkan last_from_me/last_sender utk akurat
	chats := a.GetChats()

	const topN = 14
	messages := map[string][]MessageDTO{}
	for i, c := range chats {
		if i >= topN {
			break
		}
		messages[c.ID] = a.GetMessages(c.ID)
	}

	self := eng.SelfJID()
	meName := ""
	if self != "" {
		meName = eng.ChatName(self)
	}
	if meName == "" {
		meName = "Saya"
	}

	out := map[string]interface{}{
		"chats":    chats,
		"messages": messages,
		"me":       map[string]string{"name": meName, "phone": readablePhone(eng, self)},
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readablePhone(eng *engine.Engine, jid string) string {
	if jid == "" {
		return ""
	}
	if p := eng.ReadableID(jid); p != "" {
		return p
	}
	return shortJID(jid)
}
