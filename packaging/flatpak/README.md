# Flatpak packaging

WhatsLite ships as a **single pure-Go (Gio) binary** with the whatsmeow + SQLite engine running
**in-process** — one process, one window, no IPC bridge and no separate engine binary. It runs on the
**freedesktop runtime** (GL/Wayland/X11 + libopus); libmpv (for video playback) is bundled as an extra
module. (The legacy Svelte/Wails (WebKitGTK) and Qt6/QML frontends have both been removed.)

App ID: **`io.github.muhammadmishbakhuzzuhail.WhatsLite`** (neutral name — no "WhatsApp"/Meta artwork, per
trademark rules).

## Build & install locally

```sh
# 1. Tooling + runtime (once)
sudo pacman -S --needed flatpak flatpak-builder        # or your distro's equivalent
flatpak install -y flathub \
  org.freedesktop.Platform//24.08 org.freedesktop.Sdk//24.08 \
  org.freedesktop.Sdk.Extension.golang//24.08

# 2. Build + install (from the repo root)
flatpak-builder --user --install --force-clean build-flatpak \
  packaging/flatpak/io.github.muhammadmishbakhuzzuhail.WhatsLite.yml

# 3. Run
flatpak run io.github.muhammadmishbakhuzzuhail.WhatsLite
```

Data lives in `~/.var/app/io.github.muhammadmishbakhuzzuhail.WhatsLite/` (sandboxed, separate from a
native install — you'll pair the device again the first time).

> Status: **experimental, untested on a clean machine.** Verify the build completes and the app launches
> before relying on it. The manifest's archive source points at `v0.2.0` with a placeholder sha256 — bump
> the tag + real sha256 to a release that contains the Gio UI before building. The bundled `mpv`
> module is heavy; if video playback isn't needed it can be dropped (voice still works via libopus).

## Self-hosted Flatpak repo (distribute without Flathub)

```sh
flatpak-builder --repo=repo --force-clean build-flatpak \
  packaging/flatpak/io.github.muhammadmishbakhuzzuhail.WhatsLite.yml
flatpak build-bundle repo whatslite.flatpak io.github.muhammadmishbakhuzzuhail.WhatsLite
# users: flatpak install --user whatslite.flatpak
```

Or publish the `repo/` directory (e.g. GitHub Pages) and have users `flatpak remote-add` it.

## Submitting to Flathub (stricter — later)

Flathub **forbids network access at build time**, so the manifest's `--share=network` must be removed and
replaced with vendored sources:

- **Go**: commit a vendor dir (`go mod vendor`) or generate module sources; build with `-mod=vendor`.
  (The Gio UI has no npm step, so `flatpak-node-generator` is not needed.)
- Keep the neutral app-id, the unofficial disclaimer as the **first** paragraph of the metainfo
  description (already done), and no WhatsApp/Meta trademarks or artwork.

Note: no reverse-engineered-protocol (whatsmeow) WhatsApp client has Flathub precedent — acceptance is
probable for a neutrally-named, disclaimed FOSS app but not guaranteed. The self-hosted repo above is the
reliable fallback.
