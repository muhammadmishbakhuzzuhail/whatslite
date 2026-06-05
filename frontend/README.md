# Frontend WhatsApp Lite (Svelte + Vite)

UI web yang dibungkus jadi app Linux native lewat **Wails** (WebView sistem / WebKitGTK).
Saat dev, bisa juga dibuka di browser biasa dengan **data mock**.

## Struktur

```
src/
  main.js                 bootstrap (mount App, import CSS global)
  App.svelte              layout root: Rail + Sidebar + Conversation + InfoPanel
  stores.js               state reaktif (chats, activeChatId, theme, dst)
  services/
    data.js               ⭐ SATU seam data: mock sekarang, engine Go nanti
  lib/
    data/mock.js          ⭐ DATA PALSU — edit di sini untuk ubah isi mockup
    util.js
    Rail.svelte           rail ikon kiri
    common/               Avatar, Ticks, ThemeToggle
    sidebar/              ChatsPane, ChatList, ChatRow, SearchBar, Filters,
                          SettingsPane, ProfilePane, PlaceholderPane
    chat/                 Conversation, ConvHeader, MessageList, Bubble, InfoPanel
  styles/app.css          token warna + gaya global (light/dark)
```

## Fleksibel: dua titik ubah

1. **Ubah isi tampilan** → edit `src/lib/data/mock.js` (chats, messagesByChat, me, dst).
   Bentuk objeknya sengaja dekat ke model engine nyata.
2. **Ganti sumber data mock → engine nyata** → edit **hanya** `src/services/data.js`
   (set `LIVE` + panggil `window.go.main.App.*`). Komponen TIDAK perlu diubah.

## Menjalankan

```sh
npm install
npm run dev       # dev server + hot reload  (http://localhost:5173)
npm run build     # build statis ke dist/  (di-embed Wails)
npm run preview   # serve hasil build       (http://localhost:4173)
```

Pratinjau tema/tampilan via query: `?theme=dark`, `?view=settings`, `?info=1`.

## Screenshot (loop verifikasi)

```sh
npm run preview &
firefox --headless --window-size=1200,760 \
  --screenshot /tmp/wa.png "http://localhost:4173/?theme=dark"
```

App native penuh dibangun dari root proyek: `wails build -tags webkit2_41`.
