# Flatpak packaging

A Flatpak bundles its own WebKitGTK 4.1 + glibc (via the **GNOME runtime**), so it runs on **any** distro
— including ones the prebuilt binary can't reach (Ubuntu 20.04, Debian 11, RHEL/Rocky/Alma). This is the
"works everywhere" path.

App ID: **`io.github.muhammadmishbakhuzzuhail.WhatsLite`** (neutral name — no "WhatsApp"/Meta artwork, per
trademark rules).

## Build & install locally

```sh
# 1. Tooling + runtime (once)
sudo pacman -S --needed flatpak flatpak-builder        # or your distro's equivalent
flatpak install -y flathub \
  org.gnome.Platform//48 org.gnome.Sdk//48 \
  org.freedesktop.Sdk.Extension.golang//24.08 \
  org.freedesktop.Sdk.Extension.node20//24.08

# 2. Build + install (from the repo root)
flatpak-builder --user --install --force-clean build-flatpak \
  packaging/flatpak/io.github.muhammadmishbakhuzzuhail.WhatsLite.yml

# 3. Run
flatpak run io.github.muhammadmishbakhuzzuhail.WhatsLite
```

Data lives in `~/.var/app/io.github.muhammadmishbakhuzzuhail.WhatsLite/` (sandboxed, separate from a
native install — you'll pair the device again the first time).

> Status: **experimental, untested on a clean machine.** Verify the build completes and the app launches
> before relying on it. If the GNOME runtime version `48` is unavailable, try `47` (both ship WebKitGTK
> 4.1) — bump `runtime-version` + the SDK extension `//24.08` branch accordingly.

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
- **npm**: generate `node-sources.json` with
  [`flatpak-node-generator`](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) from
  `frontend/package-lock.json`, and add it as a source.
- Keep the neutral app-id, the unofficial disclaimer as the **first** paragraph of the metainfo
  description (already done), and no WhatsApp/Meta trademarks or artwork.

Note: no reverse-engineered-protocol (whatsmeow) WhatsApp client has Flathub precedent — acceptance is
probable for a neutrally-named, disclaimed FOSS app but not guaranteed. The self-hosted repo above is the
reliable fallback.
