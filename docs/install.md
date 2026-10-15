# Installation, update and removal

Assets belong to an actually published tag at
https://github.com/mohammadhejazirad/tunnelctl/releases. No published version is
implied here. Replace VERSION with its version **without v**, ARCH with amd64
or arm64. Windows has amd64 only. Only Linux amd64 proxy is approved; other
installer targets are experimental and connection runtime is refused.

## Select an asset

All filenames have prefix `tunnelctl_VERSION_`:

| Platform | Native asset suffix | Portable suffix |
|---|---|---|
| Debian/Ubuntu | linux_ARCH.deb | linux_ARCH.tar.gz / linux_ARCH.AppImage |
| Fedora | linux_ARCH.rpm | linux_ARCH.tar.gz / linux_ARCH.AppImage |
| Arch x86_64 | linux_amd64.pkg.tar.zst | linux_amd64.tar.gz |
| Arch Linux ARM (not install-tested) | experimental linux_arm64.pkg.tar.zst / linux_arm64.PKGBUILD | linux_arm64.tar.gz |
| Alpine/musl | linux_ARCH.apk | linux_ARCH.tar.gz / AppImage payload extraction |
| macOS | darwin_ARCH.pkg | darwin_ARCH.tar.gz |
| Windows | windows_amd64.msi | windows_amd64.zip |

There is no APT/DNF/AUR/Homebrew/WinGet repository. openSUSE is not certified.
Do not infer proxy/TUN certification from installation success. Release binaries
use CGO=0 (no libc linkage); the external core has separate compatibility gates.

## Verify before installing

Obtain checksums.txt from the same tag. Verify the exact selected filename:

```sh
# Linux: print/inspect its checksum record; absence is an error.
grep '  tunnelctl_VERSION_linux_amd64.deb$' checksums.txt | sha256sum --check --strict
# macOS: compare the output to its exact record in checksums.txt.
shasum -a 256 tunnelctl_VERSION_darwin_arm64.pkg
```

```powershell
$asset = 'tunnelctl_VERSION_windows_amd64.msi'
$entry = @(Get-Content -LiteralPath .\checksums.txt | Where-Object { $_.EndsWith("  $asset") })
if ($entry.Count -ne 1) { throw 'Checksum record missing or ambiguous' }
if ((Get-FileHash -LiteralPath $asset -Algorithm SHA256).Hash -ine $entry[0].Split(' ')[0]) { throw 'Checksum mismatch' }
```

Checksums detect corruption, not independent authenticity. Verify GitHub build
provenance when available: `gh attestation verify ASSET --repo
mohammadhejazirad/tunnelctl`. Packages/MSI/pkg are unsigned; macOS is not
notarized. Do not disable platform security for blocked artifacts.

## Linux native packages

```sh
sudo apt install ./tunnelctl_VERSION_linux_amd64.deb      # Debian/Ubuntu
sudo dnf install ./tunnelctl_VERSION_linux_amd64.rpm      # Fedora
sudo pacman -U ./tunnelctl_VERSION_linux_amd64.pkg.tar.zst # Arch; not AUR
sudo apk add --allow-untrusted ./tunnelctl_VERSION_linux_amd64.apk # Verified LOCAL file only
cd /tmp
tunnelctl version --json
tunnelctl --help
```

Use arm64 on appropriate ARM targets. Only package operations require root.
Payload is `/usr/bin/tunnelctl` plus public docs/notices; no connection, host
route/DNS/firewall change, shell edit, core installation or boot service occurs.
Installation does not create a user's config. No credentials enter PATH.
deb/rpm/Arch refuse unmanaged destination collisions. apk does not reliably
abort extraction after a failed preinstall script: its dedicated hook warns and
moves the old file intact into root-private
`/var/backups/tunnelctl-unmanaged.XXXXXXXX/tunnelctl`. Review/migrate it explicitly;
removal preserves the backup. Stop sessions before update/removal; do not mix
installation methods at one destination.

For TUI clipboard import (`v`), install `wl-clipboard` on Wayland or `xclip`/`xsel`
on X11. Manual hidden paste (`Profiles` then `i`) remains available without a
helper. Clipboard content is read only after the explicit `v` action.

The generated PKGBUILD references the exact tarball and SHA-256. Save it as
PKGBUILD in a new owned build directory, run makepkg as an ordinary Arch user,
then install with pacman. It is not an AUR submission. Arch ARM64 remains an
experimental payload build, not an approved Arch Linux ARM installation.

## Manual archive installation

Inspect paths/types before extracting. Keep notices and offline docs. Per-user:

```sh
mkdir -p "$HOME/.local/lib" "$HOME/.local/bin"
mkdir "$HOME/.local/lib/tunnelctl-VERSION"
tar -xzf tunnelctl_VERSION_linux_amd64.tar.gz -C "$HOME/.local/lib/tunnelctl-VERSION"
ln -s "$HOME/.local/lib/tunnelctl-VERSION/tunnelctl" "$HOME/.local/bin/tunnelctl"
export PATH="$HOME/.local/bin:$PATH" # Current shell only, if needed.
cd /tmp
tunnelctl version --json
```

The link operation refuses an existing command. No shell startup files are
modified; if ~/.local/bin is absent from PATH, configure it explicitly using
your shell/admin's normal process. System-wide manual installs can use a
root-owned 0755 `/usr/local/bin/tunnelctl` after inspecting existing ownership.
Never manually install into `/usr/bin`, overwrite another manager's executable
or replace an active core. Use private Unix-permission-supporting storage, not
FAT/fuseblk, for secrets and detached executables.

## AppImage, without a desktop

```sh
chmod +x tunnelctl_VERSION_linux_amd64.AppImage
./tunnelctl_VERSION_linux_amd64.AppImage version --json
./tunnelctl_VERSION_linux_amd64.AppImage --help
```

Mounted execution requires working FUSE (/dev/fuse and distro permissions).
No terminal emulator is launched; Terminal=true is only desktop-entry metadata.
AppRun directly references the CLI; configuration stays in per-user XDG paths,
not the read-only mount. Direct FUSE execution is untested in this development
environment. Mounted detached supervision is not certified.

**Tested no-FUSE fallback:** extract into a stable, owned directory and execute
AppRun directly. Transient --appimage-extract-and-run sessions are explicitly refused:
upstream's extra parent has different signal/exit behavior and failed lifecycle
tests. Stable extraction uses the actual binary with normal process semantics:

```sh
mkdir "$HOME/.local/lib/tunnelctl-AppImage-VERSION"
cd "$HOME/.local/lib/tunnelctl-AppImage-VERSION"
/absolute/path/tunnelctl_VERSION_linux_amd64.AppImage --appimage-extract
./squashfs-root/AppRun version --json
# Optional explicit PATH link, without replacing an existing command:
ln -s "$HOME/.local/lib/tunnelctl-AppImage-VERSION/squashfs-root/usr/bin/tunnelctl" "$HOME/.local/bin/tunnelctl"
```

Moving that directory requires explicitly updating its link. A direct image
link must likewise target a fixed absolute image location; moving the image
breaks it and FUSE is still required. No TUN/root execution, universal Linux
support or working transient extraction lifecycle is claimed.

## Windows MSI and ZIP (experimental runtime)

Administrator PowerShell, for installation only:

```powershell
$msi = (Resolve-Path .\tunnelctl_VERSION_windows_amd64.msi).Path
$p = Start-Process msiexec.exe -ArgumentList @('/i', ('"' + $msi + '"'), '/qn', '/norestart') -Wait -PassThru
if ($p.ExitCode -notin @(0,3010)) { throw "Installer failed: $($p.ExitCode)" }
```

Open a **new ordinary PowerShell** after installation:

```powershell
Set-Location $env:TEMP
Get-Command tunnelctl
tunnelctl version --json
tunnelctl --help
```

MSI installs Program Files/tunnelctl and its machine PATH entry through the
Windows Installer Environment table: no whole-PATH replacement, duplicate entry
or private credential variables. Stable UpgradeCode identifies the family;
downgrade is refused and repair restores files. Uninstall removes only its owned
entry/payload. No per-user MSI is offered.
An independently configured machine PATH entry pointing at that destination is
refused, not claimed and later removed; migrate the old installation explicitly.
ZIP extraction alone does not edit
PATH; invoke its exe by absolute path or explicitly add its directory to your
user PATH. Windows connection/storage runtime is refused pending ACL/IPC/lifecycle
approval; no driver is shipped. Installer success does not change that boundary.

## macOS pkg (experimental runtime)

```sh
sudo installer -pkg ./tunnelctl_VERSION_darwin_arm64.pkg -target /
cd /tmp
tunnelctl version --json
tunnelctl --help
```

Use amd64 on Intel. Payload is root-owned `/Library/Application Support/tunnelctl`, command is
an owned `/usr/local/bin/tunnelctl` symlink; no shell-file changes. Preinstall
refuses unmanaged payloads, Homebrew/same-name destination collisions. No universal
binary, notarization or macOS proxy support is claimed. For policy-blocked
unsigned artifacts use reviewed source builds, not disabled Gatekeeper.

## Update and uninstall

Stop first, privately back up data ([storage](storage.md)), verify the new tag's
assets and install via the same manager. Native versions differ from binary
SemVer ([mapping](packaging.md)). No old public installer exists for local
upgrade certification; native CI uses a labeled synthetic MSI upgrade fixture.
Older schema-incompatible executables refuse new data without overwriting it.

```sh
sudo apt remove tunnelctl       # Purge also has no user-data deletion hook
sudo dnf remove tunnelctl
sudo pacman -R tunnelctl
sudo apk del tunnelctl
sudo '/Library/Application Support/tunnelctl/uninstall.sh' # macOS receipt-scoped removal
```

```powershell
$msi = (Resolve-Path .\tunnelctl_VERSION_windows_amd64.msi).Path
Start-Process msiexec.exe -ArgumentList @('/fa', ('"' + $msi + '"'), '/qn') -Wait # Repair
Start-Process msiexec.exe -ArgumentList @('/x', ('"' + $msi + '"'), '/qn') -Wait  # Uninstall
```

Manual/AppImage uninstall unlinks only your known launcher and removes only the
inspected public version directory after stopping. Never remove a broad home/XDG
root, shared bin directory or another program's core. All ordinary uninstalls
preserve profiles, secret source URLs, settings, backups and caches. Full private
deletion is a separate explicit [storage procedure](storage.md). Installers never
read users' homes or change private-file ownership. System-wide binary does not
make per-user data shared.

## Paths and command not found

Supported Linux: `tunnelctl paths --json` and doctor show nonsecret paths under
XDG_CONFIG_HOME/tunnelctl, XDG_STATE_HOME/tunnelctl and XDG_CACHE_HOME/tunnelctl,
with standard per-user fallbacks. Logs are bounded private state; transient
session records are not durable connection truth. Installation/last-profile
loading never auto-connects. Unsupported macOS/Windows runtime does not silently
share a config under the install directory.

Check `command -v tunnelctl` / PowerShell `Get-Command tunnelctl`, installer exit
code, architecture, actual PATH and any moved manual/AppImage link. Open a new
terminal after MSI. Never copy secrets into PATH or auto-edit shell startup files
to hide a failed installation.
