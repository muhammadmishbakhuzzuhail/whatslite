# Third-Party Notices

WhatsApp Lite is licensed under **GPL-3.0** (see [`LICENSE`](./LICENSE)). It bundles or links the
third-party components below. Each remains under its own license; those licenses are compatible with
distributing the combined work under the GPL (notably MPL-2.0 §3.3, which permits distribution of a larger
work under the GPL).

## Backend (Go)

| Component | License | Notes |
|---|---|---|
| [go.mau.fi/whatsmeow](https://github.com/tulir/whatsmeow) | **MPL-2.0** | WhatsApp multi-device protocol library |
| [github.com/wailsapp/wails/v2](https://github.com/wailsapp/wails) | MIT | Go ↔ system WebView shell |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | BSD-3-Clause | Pure-Go SQLite (no CGo) |
| [google.golang.org/protobuf](https://github.com/protocolbuffers/protobuf-go) | BSD-3-Clause | Protobuf runtime |
| [github.com/mdp/qrterminal/v3](https://github.com/mdp/qrterminal) | MIT | Terminal QR rendering (CLI) |
| [github.com/skip2/go-qrcode](https://github.com/skip2/go-qrcode) | MIT | QR code generation |

## Frontend (web)

| Component | License | Notes |
|---|---|---|
| [Svelte](https://github.com/sveltejs/svelte) | MIT | UI framework |
| [Vite](https://github.com/vitejs/vite) | MIT | Build tooling |
| [emoji-picker-element](https://github.com/nolanlawson/emoji-picker-element) | Apache-2.0 | Emoji & reaction picker (lazy-loaded) |

## Runtime services (not bundled; called over the network)

These are external services the app talks to at runtime. No code is bundled, but use is subject to each
provider's terms:

- **WhatsApp** (via the WhatsApp Web protocol) — see the Disclaimer in [`README.md`](./README.md); use
  violates WhatsApp's ToS and may get the linked number banned.
- **Tenor** — GIF search.
- **Google Translate** (gtx endpoint) — message translation and machine-translated UI locales.
- **Static map provider** — location-message preview thumbnails.

MPL-2.0 source for whatsmeow is available at its upstream repository linked above.
