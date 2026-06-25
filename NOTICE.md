# Third-Party Notices

WhatsLite is licensed under **GPL-3.0** (see [`LICENSE`](./LICENSE)). It bundles or links the
third-party components below. Each remains under its own license; those licenses are compatible with
distributing the combined work under the GPL (notably MPL-2.0 §3.3, which permits distribution of a larger
work under the GPL).

## Backend (Go)

| Component | License | Notes |
|---|---|---|
| [go.mau.fi/whatsmeow](https://github.com/tulir/whatsmeow) | **MPL-2.0** | WhatsApp multi-device protocol library |
| [gioui.org](https://gioui.org) (+ `gioui.org/x`) | Unlicense OR MIT | Immediate-mode GUI toolkit (UI, richtext/styledtext) |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | BSD-3-Clause | Pure-Go SQLite (no CGo) |
| [google.golang.org/protobuf](https://github.com/protocolbuffers/protobuf-go) | BSD-3-Clause | Protobuf runtime |
| [github.com/mdp/qrterminal/v3](https://github.com/mdp/qrterminal) | MIT | Terminal QR rendering (CLI) |
| [github.com/skip2/go-qrcode](https://github.com/skip2/go-qrcode) | MIT | QR code generation |
| [github.com/ebitengine/oto/v3](https://github.com/ebitengine/oto) | Apache-2.0 | Audio playback (voice notes) |
| [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg) | BSD-3-Clause | SVG icon parsing/rendering |
| [github.com/srwiley/rasterx](https://github.com/srwiley/rasterx) | BSD-3-Clause | Vector rasterizer (oksvg backend) |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) | BSD-3-Clause | Image scaling/decoding (downscale, draw) |

## Fonts & assets

| Component | License | Notes |
|---|---|---|
| [Go fonts](https://go.dev/blog/go-fonts) (`golang.org/x/image/font/gofont`) | BSD-3-Clause | Bundled UI text faces |

The color-emoji face (**Noto Color Emoji**, OFL-1.1) is **not bundled** — it is loaded from the host's
system font paths at runtime (`/usr/share/fonts/.../NotoColorEmoji.ttf`).

## Runtime services (not bundled; called over the network)

These are external services the app talks to at runtime. No code is bundled, but use is subject to each
provider's terms:

- **WhatsApp** (via the WhatsApp Web protocol) — see the Disclaimer in [`README.md`](./README.md); use
  violates WhatsApp's ToS and may get the linked number banned.
- **Tenor** — GIF search.
- **Google Translate** (gtx endpoint) — message translation and machine-translated UI locales.
- **Static map provider** — location-message preview thumbnails.

MPL-2.0 source for whatsmeow is available at its upstream repository linked above.
