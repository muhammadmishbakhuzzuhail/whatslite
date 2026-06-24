# Contributing

Thanks for your interest in contributing! This project is a lightweight WhatsApp desktop client for Linux
(Go + whatsmeow + Gio, single in-process binary). Start by reading the [README](./README.md), [`PRODUCT-BRIEF.md`](./PRODUCT-BRIEF.md),
and [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) to understand the project's direction and architecture.

> ⚠️ **Read the [ToS disclaimer in the README](./README.md#-disclaimer-read-first) first.** This project uses
> WhatsApp's unofficial Web protocol. Test numbers are **at risk of being banned**. Use a spare number while developing.

## Philosophy: stay *lean*

This project's main differentiator is being **lightweight**. Before adding a feature or dependency, ask yourself:

- Can it be done without a new dependency? (the binary is currently ~25 MB — keep it small)
- Does it add significant runtime weight (RAM/disk)?
- Does it load large data into the UI? (don't decode full base64 media into widgets — use files/cache)

PRs that add heavy dependencies (e.g. large ML/WASM, Electron-isms) will likely be rejected unless the
payoff is worth it. For context: ONNX background removal was once dropped (binary went 50→25 MB).

## Prerequisites & build

See [README §Build prerequisites](./README.md#build-prerequisites-linux). In short (Arch/CachyOS):

```sh
sudo pacman -S --needed go mesa wayland libxkbcommon libx11 libxcursor libxfixes vulkan-icd-loader opus mpv pkgconf
```

```sh
# run the app:
go run ./cmd/whatslite-gio
# release build:
go build -o whatslite-gio ./cmd/whatslite-gio
```

## Before opening a PR

Run all of these (the same as CI in `.github/workflows/build.yml`):

```sh
go vet ./...
go test ./...
go build ./cmd/whatslite-gio   # must succeed
go build ./cmd/gio-shot        # headless render tool — must succeed
```

- **No dead code.** `go vet` must be clean — remove unused code.
- **Don't commit secrets.** API keys, `*.db`, binaries, and `real-data.json` are already in `.gitignore` — keep it that way.
- **Respect whatsmeow's synchronous handlers.** Heavy DB work in an event handler must be offloaded to the
  `a.bg()` queue — handlers run on the socket loop, so blocking means the websocket drops.

## Code style

- **Go**: follow `gofmt`/`go vet`. Keep comments consistent with the surrounding code.
- **Gio UI**: immediate-mode (`internal/gioui`). Match the idioms and comment density of nearby files;
  reuse the shared theme/widget helpers rather than hardcoding colors or layout constants.
- **Commits**: Conventional Commits (`feat:`, `fix:`, `perf:`, `chore:`, `refactor:`, `docs:`).

## Reporting bugs / requesting features

Use the [issue templates](./.github/ISSUE_TEMPLATE/). For security vulnerabilities, **do not** open a public issue —
see [`SECURITY.md`](./SECURITY.md).

## License

By contributing, you agree that your contributions are licensed under **GPL-3.0** (see [`LICENSE`](./LICENSE)).
