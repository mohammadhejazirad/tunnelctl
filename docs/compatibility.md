# Compatibility status

Machine-readable records are in ../compatibility.json. Source 0.2.0-dev, storage
schema 3, JSON envelope schema 1, Go 1.27.2, Cobra 1.10.1, Bubble Tea 2.1.0,
Lip Gloss 2.0.6 and sing-box 1.14.3. All modules/checksums are locked. Xray remains
unsupported. Linux proxy, subscriptions, exec/detached and TUI have local evidence.

Linux amd64 proxy was tested on Ubuntu 26.04.1 LTS, kernel 7.0.0-38-generic,
UID 1000, local Linux filesystems and procfs with permitted loopback sockets.
No systemd is used. A local filesystem supporting Unix permissions, atomic rename,
fsync and flock is required. procfs and SO_PEERCRED are required for the Linux
lifecycle implementation. The minimum supported kernel and broader distribution
matrix have not been established.

The current source workspace is on a fuseblk mount that presents nobody ownership
and 0777 modes. It is usable for source/build artifacts but was rejected for the
core and private data. Tests use private /tmp directories with real Unix permissions.

Runtime commands explicitly reject non-Linux platforms; core approval is currently
linux-amd64 only. macOS/Windows builds do not implement native private storage or
session ownership. Version/help remain usable in compiled binaries. Linux arm64
requires an independently approved artifact and native tests before support.
PowerShell, macOS, containers, WSL and other distributions are not functionally
certified. TUN is unsupported everywhere in this phase.

| Distribution/environment | Exact native version | App/core proxy test | TUN/service |
|---|---|---|---|
| Ubuntu amd64 | 26.04.1, kernel 7.0.0-38-generic | Passed | Unsupported |
| Ubuntu CI | Runner ubuntu-24.04 | Configured; remote run pending | Unsupported |
| Debian | None | Not tested | Unsupported |
| Fedora | None | Not tested | Unsupported |
| Arch | None | Not tested | Unsupported |
| openSUSE | None | Not tested | Unsupported |
| Alpine/musl | None | Not tested | Unsupported |
| WSL / container / restricted VPS | None | Not certified | Unsupported |

App release CGO=0 does not by itself prove musl/core compatibility. Proxy does not
integrate with systemd, NetworkManager or host DNS managers; absence of an init
dependency is not certification of every distribution. Feature checks matter more
than distro names. Minimum kernel version is unestablished. Windows PowerShell
completion is generated but native commands beyond help/version are unsupported.

## Native package evidence (2026-10-15)

These **amd64 container installation tests** are separate from the host/core
proxy table above. Each ran offline after image download: unmanaged collision,
install from /tmp, command lookup, help/version, root ownership/modes, native
version ordering where available, removal and preservation of synthetic user data.

| Image / exact release | Format | Local evidence |
|---|---|---|
| Ubuntu 24.04.5 LTS | deb | Passed |
| Debian 13.7 (trixie) | deb | Passed |
| Fedora 43 Container Image | rpm | Passed (merged /usr/sbin path alias accounted for) |
| Arch 20261011.0.609700 | pkg.tar.zst | Passed; generated PKGBUILD built as ordinary user and resulting package installed/removed |
| Alpine 3.22.6 (musl) | apk | Passed; unmanaged backup/warning tested |

Image digests are pinned in packaging.yml. These tests do not certify TUN,
NetworkManager/systemd/DNS integration, musl core compatibility or host kernel
support. No host package was installed/removed. ARM64 native containers,
macOS pkg, Windows MSI and PowerShell installer tests are configured in GitHub
Actions but not locally executed. Arch ARM64 is only an experimental payload
build; no approved Arch Linux ARM native installation environment is provided.
openSUSE, WSL, service and all TUN combinations remain uncertified.

AppImage amd64: stable extracted payload passed real proxy/HTTP/SOCKS/TLS,
duration, Ctrl+C/SIGTERM/SIGHUP, core/supervisor crash and detached/exec tests.
Temporary --appimage-extract-and-run failed directed-signal/lifecycle tests;
shared application services now refuse sessions in that mode. Metadata/import
remain testable there. FUSE-mounted execution cannot be tested here (/dev/fuse
absent); source confirms exec-based AppRun but this is not runtime evidence.
See [packaging contract](packaging.md).
