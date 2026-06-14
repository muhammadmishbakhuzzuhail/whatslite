# WhatsApp Lite frontend (Svelte + Vite)

The web UI, packaged into a native Linux app via **Wails** (system WebView / WebKitGTK).
During development it can also be opened in a regular browser with **mock data**.

## Structure

```
src/
  main.js                 bootstrap (mount App, import global CSS)
  App.svelte              root layout: Rail + Sidebar + Conversation + InfoPanel
  stores.js               reactive state (chats, activeChatId, theme, etc.)
  services/
    data.js               ⭐ the SINGLE data seam: live Go engine, or mock as a fallback
  lib/
    data/mock.js          ⭐ MOCK DATA — edit here to change the mockup contents
    util.js
    Rail.svelte           left icon rail
    common/               Avatar, Ticks, ThemeToggle
    sidebar/              ChatsPane, ChatList, ChatRow, SearchBar, Filters,
                          SettingsPane, ProfilePane, PlaceholderPane
    chat/                 Conversation, ConvHeader, MessageList, Bubble, InfoPanel
  styles/app.css          color tokens + global styles (light/dark)
```

## Two points of change

1. **Change what's displayed** → edit `src/lib/data/mock.js` (chats, messagesByChat, me, etc.).
   The object shapes are intentionally close to the real engine model.
2. **Switch data source mock ↔ real engine** → edit **only** `src/services/data.js`. When the app runs
   inside Wails, `window.go.main.App.*` is present and `LIVE` is on; in a plain browser it falls back to
   mock data. Components don't need to change.

## Running

```sh
npm install
npm run dev       # dev server + hot reload  (http://localhost:5173)
npm run build     # static build to dist/    (embedded by Wails)
npm run preview   # serve the build           (http://localhost:4173)
```

Preview theme/views via query params: `?theme=dark`, `?view=settings`, `?info=1`.

## Screenshot (verification loop)

```sh
npm run preview &
firefox --headless --window-size=1200,760 \
  --screenshot /tmp/wa.png "http://localhost:4173/?theme=dark"
```

The full native app is built from the project root: `wails build -tags webkit2_41`.
