# AUR packaging

Two PKGBUILDs (each is its own AUR package — one directory each):

- **`whatslite/`** — `whatslite`: builds a **tagged release** from source (stable). Tracks releases.
- **`whatslite-git/`** — `whatslite-git`: builds the **latest commit** from source. Tracks `main`.

They declare each other in `conflicts`, so only one can be installed at a time. Both build natively
(Wails uses cgo + WebKitGTK and cannot cross-compile) and stamp the version into `main.version` via
`-ldflags`.

## Try locally (no AUR account needed)

```sh
cd packaging/aur/whatslite      # or whatslite-git
makepkg -si
```

Pulls `webkit2gtk-4.1` + `gtk3` (runtime) and `go`/`nodejs`/`npm`/`git` (build), compiles, installs
`whatslite` to `/usr/bin`, plus the `.desktop` entry and icon.

## Publish to the AUR

You need an AUR account with an SSH key (https://aur.archlinux.org → My Account → SSH keys). Publish each
package from its own directory:

```sh
cd packaging/aur/whatslite               # (repeat for whatslite-git)
makepkg --printsrcinfo > .SRCINFO        # required by the AUR
git clone ssh://aur@aur.archlinux.org/whatslite.git /tmp/aur-whatslite
cp PKGBUILD .SRCINFO /tmp/aur-whatslite/
cd /tmp/aur-whatslite && git add PKGBUILD .SRCINFO && git commit -m "Initial import" && git push
```

After that: `yay -S whatslite` (stable) or `yay -S whatslite-git` (latest).

## Cutting a new release (updating the stable package)

1. Tag + GitHub release upstream (see repo CHANGELOG workflow).
2. In `whatslite/PKGBUILD`: bump `pkgver` to the new version, reset `pkgrel=1`.
3. Refresh the source checksum:
   ```sh
   cd packaging/aur/whatslite
   updpkgsums                            # rewrites sha256sums from the new tarball
   makepkg --printsrcinfo > .SRCINFO
   ```
4. Commit + push to the AUR repo.

> Note: `sha256sums` here is the hash of GitHub's auto-generated
> `archive/vX.Y.Z.tar.gz`. These bytes are stable in practice but have rarely changed; if `makepkg`
> reports a checksum mismatch, rerun `updpkgsums`.
