// Command walite-cli: mode CLI headless untuk uji engine (pairing QR di
// terminal + log pesan masuk). Berguna untuk debugging tanpa GUI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mdp/qrterminal/v3"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/engine"
)

func main() {
	ctx := context.Background()

	dataDir, err := engine.DefaultDataDir()
	if err != nil {
		fatal("tidak bisa menyiapkan direktori data: %v", err)
	}
	dbPath := filepath.Join(dataDir, "whatslite.db")

	debug := os.Getenv("WALITE_DEBUG") != ""
	eng, err := engine.New(ctx, dbPath, debug)
	if err != nil {
		fatal("gagal inisialisasi engine: %v", err)
	}

	eng.OnMessage(func(m engine.IncomingMessage) {
		name := m.PushName
		if name == "" {
			name = m.Sender
		}
		text := m.Text
		if text == "" {
			text = "[pesan non-teks]"
		}
		fmt.Printf("\n📩 [%s] %s: %s\n", m.Chat, name, text)
	})

	qr, err := eng.Start(ctx)
	if err != nil {
		fatal("gagal connect: %v", err)
	}

	if qr != nil {
		fmt.Println("Scan QR ini di WhatsApp > Perangkat Tertaut (Linked Devices):")
		for evt := range qr {
			switch evt.Event {
			case "code":
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			case "success":
				fmt.Println("✅ Login berhasil. Menunggu pesan...")
			case "timeout":
				fmt.Println("⌛ QR kedaluwarsa. Jalankan ulang aplikasi.")
			case "error":
				fmt.Printf("❌ Error pairing: %v\n", evt.Err)
			default:
				fmt.Println("Event login:", evt.Event)
			}
		}
	} else {
		fmt.Println("✅ Sudah login. Menunggu pesan... (Ctrl-C untuk keluar)")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("\nMenutup koneksi...")
	eng.Stop()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fatal: "+format+"\n", args...)
	os.Exit(1)
}
