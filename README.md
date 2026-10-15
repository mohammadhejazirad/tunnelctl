# tunnelctl

A terminal-only Go client for existing proxy servers. Scriptable CLI and keyboard
TUI; no browser, dashboard or desktop session required.

Source version: **0.2.0-dev**. Release dry-run candidate: **v0.2.0-rc.1**, not a
published release. Repository: https://github.com/mohammadhejazirad/tunnelctl.
No download badge implies assets have already been published.

## Features and limits

- Real loopback HTTP/SOCKS5 proxy, foreground or explicit detached supervisor.
- Duration, owned-process stop, private status and bounded state events.
- Clipboard/manual import for SS, VLESS, VMess and Trojan share links; profile
  selection, rename, removal and redacted JSON.
- Durable shared CLI/TUI preferences, tags/favorites and protected migration backups.
- Transactional HTTPS subscriptions: preview, last-good preservation, stable IDs,
  conditional requests and explicit deletion approval.
- Isolated child `exec`, shell completion and Bubble Tea/Lip Gloss keyboard TUI.
- SemVer metadata, schema migration, native packaging, checksums/module SBOM and
  fully gated automatic tag releases (draft/upload verification before publication).

Supported share links are **Shadowsocks SIP002, VLESS, VMess and Trojan**. The
sing-box adapter supports TCP plus WebSocket, gRPC, HTTP and HTTP Upgrade where the
protocol allows them, with verified TLS and Reality. Insecure certificate bypass,
plugins, XHTTP/KCP/QUIC share transports and arbitrary core/Clash JSON/YAML remain
rejected. Feeds contain newline links or Base64 envelopes. Import is not connection
verification. **TUN, UDP, services, automatic updates and scheduled refresh are not
implemented.** No fake counters or kill-switch controls exist.

Proxy affects only applications explicitly using it. TUN would change host
networking and requires separate privilege, ownership and recovery gates.

## Platform support

| Target | Build | Native evidence | Proxy | TUN/service |
|---|---|---|---|---|
| Linux amd64 | Tested | Ubuntu 26.04.1, kernel 7.0.0-38-generic | Tested | Unsupported |
| Linux arm64 | Compile-only | Package/payload gates passed; no approved core test | Unsupported | Unsupported |
| macOS amd64 / arm64 | Compile-only | Installer lifecycle passed; runtime storage pending | Unsupported | Unsupported |
| Windows amd64 | Compile-only | MSI rerun and ACL/IPC runtime gates pending | Unsupported | Unsupported |
| Windows arm64 | Not built | Not tested | Unsupported | Unsupported |

Package installation/help/version/collision/removal have additionally been tested
in Ubuntu 24.04, Debian 13, Fedora 43, Arch (pinned container) and Alpine 3.22/musl
on amd64. Remote CI also passed Linux arm64 payload/package checks and native pkg
lifecycle tests on macOS amd64/arm64. The Windows MSI source fix still requires a
remote Windows rerun. This does **not** certify non-Linux core/networking behavior.
openSUSE remains untested. See [matrix](docs/compatibility.md),
[packaging evidence](docs/packaging-checklist.md) and [tests](docs/testing.md).

## Installation and prerequisites

Linux proxy requires an ordinary user, procfs, SO_PEERCRED Unix sockets, flock,
loopback TCP and private local storage supporting Unix modes, fsync and atomic
rename. No root or systemd required. Release builds use CGO=0; C/CGO is required
only for race testing.

Core is **user-installed separately**: exact approved sing-box **1.14.3**
linux-amd64 digest. No core is bundled or automatically downloaded. Follow
[core verification](docs/core-versions.md); never replace an active core.

When a release exists, download its assets/checksums from the same tag at
https://github.com/mohammadhejazirad/tunnelctl/releases. Until then build from
source. VERSION below means the published version without `v`, not a promised
release. The complete [installation guide](docs/install.md) covers checksums,
PATH, system-wide/per-user installs, updates, removal and collision behavior.

| Target | Installer | Portable |
|---|---|---|
| Linux amd64 / arm64 | `.deb`, `.rpm`, `.pkg.tar.zst` + `PKGBUILD`, `.apk` | `.tar.gz`, `.AppImage` |
| macOS amd64 / arm64 (experimental runtime) | `.pkg` | `.tar.gz` |
| Windows amd64 (experimental runtime) | `.msi` | `.zip` |

Names: `tunnelctl_VERSION_OS_ARCH.EXT` (macOS OS name is `darwin`). Arch ARM64
is payload-smoke-only, not an approved native installation. No Windows arm64,
universal binary or APT/DNF/AUR/Homebrew/WinGet repository is offered.

```sh
sudo apt install ./tunnelctl_VERSION_linux_amd64.deb       # Debian/Ubuntu
sudo dnf install ./tunnelctl_VERSION_linux_amd64.rpm       # Fedora
sudo pacman -U ./tunnelctl_VERSION_linux_amd64.pkg.tar.zst # Arch
sudo apk add --allow-untrusted ./tunnelctl_VERSION_linux_amd64.apk # Verified local unsigned file
cd /tmp
tunnelctl version --json
```

Native packages own `/usr/bin/tunnelctl`. macOS pkg owns
`/Library/Application Support/tunnelctl` and a checked `/usr/local/bin/tunnelctl` symlink;
MSI owns Program Files/tunnelctl and its machine PATH entry. All leave per-user
data/private cores alone and never connect, change networking or enable services.
Alpine warns and privately backs up unmanaged collisions because apk cannot
reliably abort unpacking after a script failure; other native packages refuse.

Linux per-user archive installation (example candidate only, not published):

```sh
sha256sum --check checksums.txt
# Inspect paths/types before extracting a downloaded archive.
tar -tzf tunnelctl_0.2.0-rc.1_linux_amd64.tar.gz
mkdir -p "$HOME/.local/lib" "$HOME/.local/bin"
mkdir -m 700 "$HOME/.local/lib/tunnelctl-0.2.0-rc.1"
tar -xzf tunnelctl_0.2.0-rc.1_linux_amd64.tar.gz -C "$HOME/.local/lib/tunnelctl-0.2.0-rc.1"
ln -s "$HOME/.local/lib/tunnelctl-0.2.0-rc.1/tunnelctl" "$HOME/.local/bin/tunnelctl"
"$HOME/.local/bin/tunnelctl" version --json
```

Use real Unix permissions: FAT/fuseblk drives are unsuitable for secrets, core or
detached executables. Installation only copies files; no connection, host network
change or boot persistence. Archives contain notices, dependency license texts,
offline docs and `SUPPORT.txt`.

AppImage is terminal-only, no emulator/desktop required. FUSE mounted execution
is untested here; the documented stable extraction fallback is tested. Do not use
transient `--appimage-extract-and-run` for sessions: its lifecycle differs. Never
run AppImage/TUI as root. See [FUSE, fallback and PATH links](docs/install.md#appimage-without-a-desktop).

macOS/Windows installers and archives are **experimental runtime builds**, not
supported VPN clients. The native workflows exercise installation/PATH/lifecycle,
not connection support. macOS (choose amd64 on Intel):

```sh
shasum -a 256 tunnelctl_VERSION_darwin_arm64.pkg
sudo installer -pkg ./tunnelctl_VERSION_darwin_arm64.pkg -target /
cd /tmp
tunnelctl version --json
```

```powershell
Get-FileHash .\tunnelctl_VERSION_windows_amd64.msi -Algorithm SHA256
# After verifying checksum, Administrator PowerShell for installation only:
$msi = (Resolve-Path .\tunnelctl_VERSION_windows_amd64.msi).Path
$p = Start-Process msiexec.exe -ArgumentList @('/i', ('"' + $msi + '"'), '/qn', '/norestart') -Wait -PassThru
if ($p.ExitCode -notin @(0,3010)) { throw "Installer failed: $($p.ExitCode)" }
# Open a NEW ordinary PowerShell to refresh PATH:
Set-Location $env:TEMP
Get-Command tunnelctl
tunnelctl version --json
# init/run are unsupported on Windows; do not deploy as a Windows VPN.
```

Checksums detect corruption, not independent authenticity. Installers are unsigned
and macOS not notarized; do not disable system security to bypass policy. The
release workflow requests GitHub build-provenance attestations. Once available, verify an
archive with `gh attestation verify ARCHIVE --repo mohammadhejazirad/tunnelctl`.
No local dry-run attestation/signing key is claimed.

## Quick start

Replace the core path with your protected, verified installation. Place the share
link in an owned `0600` regular file; never pass secret links in arguments.

For the easiest interactive path, copy one share link or HTTPS subscription URL,
open `tunnelctl`, then press `v`. The app reads the system clipboard on that explicit
keypress, detects the content and imports it without displaying credentials.

```sh
tunnelctl init
tunnelctl profile import --file /absolute/private/profile.txt
# Or: tunnelctl profile import --stdin < /absolute/private/profile.txt
tunnelctl profile list --json
tunnelctl profile show PROFILE_ID --redacted
tunnelctl profile select PROFILE_ID
tunnelctl --core /absolute/path/to/sing-box profile check PROFILE_ID
tunnelctl --core /absolute/path/to/sing-box core verify --json
tunnelctl --core /absolute/path/to/sing-box doctor --json
tunnelctl --core /absolute/path/to/sing-box run --duration 10m
```

`run` listens on `127.0.0.1:2080`; `--listen-port` selects another unprivileged
port; `--profile` overrides selection. Duration starts after readiness, with up to
six seconds additional cleanup. Status separates process running, listener ready
and connection tested; it never invents an upstream probe.

From a second terminal, same user/storage:

```sh
tunnelctl status --json
curl --proxy socks5h://127.0.0.1:2080 https://example.com/
tunnelctl probe --target https://example.com/ --json
tunnelctl stop
tunnelctl stop  # Safe to repeat.
```

This request contacts example.com only when explicitly executed. Loopback is
reachable by other users. On shared systems use `--proxy-auth-file
/private/listener-auth.json` for `run`/`start`; see
[synthetic format](examples/proxy-auth.json). Keep client credentials out of argv.

`probe` sends one TLS-verified HTTPS request through the matching owned session;
it reads at most 4 KiB, follows no redirects and never persists a connection claim.
Use the same `--proxy-auth-file` if configured. Its destination must have no
credentials/query/fragment. `profile check` validates config without opening a
listener. Both diagnostics are also available in the menu.

## Profiles and subscriptions

```sh
tunnelctl profile rename PROFILE_ID 'My server'
tunnelctl profile remove INACTIVE_PROFILE_ID
tunnelctl subscription add --name personal --url-stdin < /private/subscription-url.txt
tunnelctl subscription list --json
tunnelctl subscription show personal
tunnelctl subscription update personal --preview --json
tunnelctl subscription update personal --if-preview PREVIEW_TOKEN
# Only after reviewing removals:
tunnelctl subscription update personal --accept-removals --if-preview PREVIEW_TOKEN
tunnelctl subscription rename personal work
tunnelctl subscription remove work
```

Add does not download/connect. Refresh is direct HTTPS with TLS verification,
no environment proxy or converter. Private sources need explicit narrow
`--allow-private-cidr`; controlled CA trust uses `--ca-file`, not insecure TLS.
Empty/broken refresh preserves healthy data. Removing a source **retains profiles
as standalone**. Updates never switch an active session. See
[formats, limits and transactional semantics](docs/subscriptions.md).

## Detached, temporary and child execution

```sh
tunnelctl --core /absolute/path/to/sing-box start --duration 10m
tunnelctl status
tunnelctl logs --json
tunnelctl stop
tunnelctl --core /absolute/path/to/sing-box exec --profile PROFILE_ID -- curl --fail https://example.com/
```

`start` explicitly detaches; terminal closure/TUI quit leave it running, but duration
still applies. No boot daemon is installed. `exec` creates an isolated ephemeral
proxy and sets only its direct child's environment. Programs may ignore variables.
It never stops/replaces an unrelated session. Child exit codes survive successful
cleanup; code 7 overrides incomplete cleanup. With `--json`, child output goes to
stderr and stdout contains one result. Arbitrary daemonized descendants are not
supervised/certified. The ephemeral child listener is unauthenticated loopback.

## TUI, status, logs and completion

```sh
tunnelctl --core /absolute/path/to/sing-box  # Interactive: open Main menu.
tunnelctl --core /absolute/path/to/sing-box tui
tunnelctl --no-color tui
tunnelctl completion bash
tunnelctl completion powershell
```

No-argument TTY startup opens Main menu; redirected startup returns help/JSON
guidance immediately. Arrows/j/k and Enter navigate; Esc cancels/returns; `?`
opens Help. Press `v` anywhere in the main UI to read and auto-detect the clipboard.
Profiles: `i` hidden manual paste, `f` private file, `/` search,
Enter details/select/config-check, `r` rename, `d` confirmed removal.
Subscriptions: `a` hidden add, `p/u` preview/update, `r` rename, `d` remove/retain.
Connect/Settings configure port, duration, authentication and an optional explicit
HTTPS probe. One draft and explicit **Save** persist all settings for CLI and TUI;
Esc discards it. Defaults never reconnect or modify an active session. All menu
starts are truly **detached**.
`x` confirms session-bound stop; `q` checks lifetime and offers leave running,
disconnect-and-exit or return. Esc/Ctrl+C cancels in-flight work without freezing
the UI; exit waits for cleanup. Read [menu guide](docs/tui.md) for all flows.
No mouse, Nerd Font or emoji required. Small terminals get a compact view.
`NO_COLOR`, `--plain`, `--no-color` control styling. `TERM=dumb`, redirected I/O,
`--json` and `--non-interactive` refuse TUI with CLI guidance.

Private real samples are **development-only and explicitly opt-in** through a
separate bounded Go harness. They are never read by ordinary startup/tests/CI,
packaged, displayed or used as public fixtures. See
[private test activation, anonymous stages and safety](docs/private-testing.md).
Unsupported options and raw core configurations remain rejected; successful
download or import is not proof of connectivity.

`logs` returns at most 256 **state events from the last session**, not raw core
logs/counters; `--follow` is unavailable. Use `doctor` and `core verify` for
storage/core checks; neither runs an unsolicited upstream request.

## Stop, recovery, service, update and uninstall

Ctrl+C/SIGTERM/SIGHUP gracefully stop foreground operation, then escalate for an
unresponsive owned core. Linux parent-death handling stops the core on supervisor
death. Status detects stale metadata; repeat stop for exact owned cleanup. Never
kill by process name or delete a whole XDG root. Offline
[recovery](docs/recovery.md) ships with archives.

**Service install/uninstall unsupported**; no systemd/launchd/Windows boot service.
Manual update: stop, privately back up data, verify new archive and replace only
the executable. Keep the old binary; schema downgrade needs a compatible backup.
No updater moves stable users to prereleases. Uninstall: stop using the installed
binary, remove that exact executable, preserve profiles unless explicitly purging
their specific files. See [installation lifecycle](docs/install.md).

## Files, troubleshooting, SSH/WSL/container

Linux uses XDG config/state/cache/runtime paths; runtime fallback is under state.
`--data-dir /private/trial` overrides all roots; pass it consistently. Directories
are `0700`, secret files `0600`. Schema-3 `profiles.json` stores complete supported
connection parameters/credentials, source URLs/ownership, names/tags/favorites,
selection and application preferences atomically. This is permission-protected
plaintext, not encryption at rest. `init` never resets existing user data.

```sh
tunnelctl paths --json
tunnelctl settings show --json
tunnelctl settings set listener-port 2081
tunnelctl settings set duration 10m
tunnelctl settings set color false
tunnelctl profile metadata PROFILE_ID --tags work,preferred --favorite
# Resets only preferences; retains profiles, subscriptions and selection:
tunnelctl settings reset --yes
```

Explicit run/start flags override saved defaults for that invocation only. Loading
saved mode/selection never auto-connects. Schema 1/2 migrate on a successful write
only after an exact private backup; corrupt/newer files are refused without reset.
Ordinary archive upgrade/uninstall preserves data outside the installation.
See [storage paths, Save/concurrency, backup, migration and explicit full removal](docs/storage.md)
and [versioning](docs/versioning.md). Native macOS/Windows storage/ACL is still
unsupported; compile-only assets do not imply persistence support.

- Port conflict: choose another port, do not kill an unknown process.
- Permission refusal: use owned private files on a suitable filesystem; do not
  weaken permissions.
- Core/hash refusal: verify the exact approved artifact.
- DNS: endpoint bootstrap uses local resolution; host DNS is unchanged.
- TLS/SSRF refusal: fix source trust/DNS/policy, not certificate verification.
- TUN: unsupported; use explicit proxy configuration. No host recovery command
  exists because this release never owns routes/DNS/firewall rules.

Proxy over SSH does not change management routes. TUN over SSH is unsupported.
VPS/container/WSL require permitted procfs, private IPC/storage and loopback;
no native WSL/container certification is claimed. glibc/musl, init and resolver
manager combinations have separate pending tests.

## Source build, tests and release policy

Use Go **1.27.2**; dependencies/checksums are locked in `go.mod`/`go.sum`.

```sh
go mod download
CGO_ENABLED=0 go build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl
go vet ./...
go test ./...
CGO_ENABLED=1 go test -race ./...  # Supported C compiler required.
make integration CORE=/absolute/path/to/approved/sing-box
make race CORE=/absolute/path/to/approved/sing-box
go run ./cmd/audit
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
govulncheck ./...
bash packaging/tools.sh /tmp/tunnelctl-package-tools
export PATH="/tmp/tunnelctl-package-tools:$PATH"
go run ./cmd/packaging --tag v0.2.0-rc.1 --dry-run --out dist/packages
bash packaging/appimage/build.sh dist/packages /tmp/tunnelctl-package-tools 0.2.0-rc.1
bash packaging/test-archives.sh dist/packages/assets v0.2.0-rc.1
```

Without explicit binary/core variables, real acceptance tests **skip**: not evidence
of connection success. See [tests](docs/testing.md), [manifest](compatibility.json),
[architecture](docs/architecture.md), [CLI](docs/cli.md),
[releasing](docs/releasing.md), [changelog](CHANGELOG.md) and
[execution checklist](docs/execution-checklist.md).

Pre-1.0 minor releases may change APIs with documented migrations; patches should
retain compatibility. App, output/storage schema and core versions are independent.
SemVer tags, including prereleases, drive release metadata. Workflow gates all
targets, creates/verifies a complete draft, then **automatically publishes** only
on a valid pushed tag. Manual dispatch is dry-run only. Missing required installers
block publication; existing releases/assets are never overwritten. Read
[tagging, signing setup and failed-draft recovery](docs/releasing.md).
Full product completion still requires separately gated TUN/native/service/backend
work. See [versioning](docs/versioning.md).

## Security and license

No telemetry, global proxy changes, shell-file edits, host route/DNS/firewall
changes or automatic privileged installation. Not a full-tunnel VPN or universal
DNS-leak/kill-switch guarantee; no independent security audit. See
[security boundary](docs/security.md) and [private reporting](SECURITY.md).
Never post live links/passwords/keys/configs. Source: Apache-2.0; external sing-box
is separately distributed under its upstream license, not bundled. Dependency
license texts accompany archives.
