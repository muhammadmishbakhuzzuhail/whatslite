# Flatpak packaging

WhatsLite ships its native **Qt6/QML** UI plus the headless Go engine on the **KDE runtime** (which bundles
Qt6 + QtQuick Controls + Qt5Compat + qt6-svg), so it runs on **any** distro — including ones the prebuilt
binary can't reach (Ubuntu 20.04, Debian 11, RHEL/Rocky/Alma). This is the "works everywhere" path. The
legacy Svelte/Wails (WebKitGTK) UI stays in the repo for rollback but is no longer the packaged frontend.

The bundle installs two binaries launched together by the `whatslite` wrapper: `whatslite-engine`
(whatsmeow + SQLite, serves the NDJSON bridge) and `walite-qt` (the Qt UI, connects to `bridge.sock`).

App ID: **`io.github.muhammadmishbakhuzzuhail.WhatsLite`** (neutral name — no "WhatsApp"/Meta artwork, per
trademark rules).

## Build & install locally

```sh
# 1. Tooling + runtime (once)
sudo pacman -S --needed flatpak flatpak-builder        # or your distro's equivalent
flatpak install -y flathub \
  org.kde.Platform//6.8 org.kde.Sdk//6.8 \
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
> before relying on it. If KDE runtime `6.8` is unavailable, try `6.7` — bump `runtime-version` + the
> matching `org.kde.Sdk` branch accordingly. The manifest's archive source still points at `v0.1.0`, which
> predates `qt-app/`; bump the tag + sha256 to a release that contains the Qt frontend before building.

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
  (The Qt frontend has no npm step, so `flatpak-node-generator` is no longer needed.)
- Keep the neutral app-id, the unofficial disclaimer as the **first** paragraph of the metainfo
  description (already done), and no WhatsApp/Meta trademarks or artwork.

Note: no reverse-engineered-protocol (whatsmeow) WhatsApp client has Flathub precedent — acceptance is
probable for a neutrally-named, disclaimed FOSS app but not guaranteed. The self-hosted repo above is the
reliable fallback.
