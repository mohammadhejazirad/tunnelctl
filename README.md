# tunnelctl

A terminal-only Go client for existing proxy servers. Scriptable CLI and keyboard
TUI; no browser, dashboard or desktop session required.

Source version: **0.2.0-dev**. Release dry-run candidate: **v0.2.0-rc.1**, not a
published release. Repository: https://github.com/mohammadhejazirad/tunnelctl.
No download badge implies assets have already been published.

## Features and limits

- Real loopback HTTP/SOCKS5 proxy, foreground or explicit detached supervisor.
- Duration, owned-process stop, private status and bounded state events.
- Private link import; profile selection, rename, removal and redacted JSON.
- Transactional HTTPS subscriptions: preview, last-good preservation, stable IDs,
  conditional requests and explicit deletion approval.
- Isolated child `exec`, shell completion and Bubble Tea/Lip Gloss keyboard TUI.
- SemVer metadata, schema migration, archive/checksum/SBOM packaging and draft CI.

Only **Shadowsocks SIP002 over TCP**, without plugins, is supported. Methods:
`aes-128-gcm`, `aes-256-gcm`, `chacha20-ietf-poly1305`. Feeds contain newline links
or Base64 envelopes, not arbitrary core/Clash JSON/YAML. Import is not connection
verification. **TUN, UDP, Xray, services, automatic updates and scheduled refresh
are not implemented.** No fake counters or kill-switch controls exist.

Proxy affects only applications explicitly using it. TUN would change host
networking and requires separate privilege, ownership and recovery gates.

## Platform support

| Target | Build | Native evidence | Proxy | TUN/service |
|---|---|---|---|---|
| Linux amd64 | Tested | Ubuntu 26.04.1, kernel 7.0.0-38-generic | Tested | Unsupported |
| Linux arm64 | Compile-only | No approved core/native test | Unsupported | Unsupported |
| macOS amd64 / arm64 | Compile-only | Storage/lifecycle pending | Unsupported | Unsupported |
| Windows amd64 | Compile-only | ACL/IPC/lifecycle pending | Unsupported | Unsupported |
| Windows arm64 | Not built | Not tested | Unsupported | Unsupported |

Debian, Fedora, Arch, openSUSE and Alpine/musl are **not yet tested**. Configuring a
CI runner does not prove it has run. See [matrix](docs/compatibility.md) and
[actual evidence](docs/testing.md).

## Installation and prerequisites

Linux proxy requires an ordinary user, procfs, SO_PEERCRED Unix sockets, flock,
loopback TCP and private local storage supporting Unix modes, fsync and atomic
rename. No root or systemd required. Release builds use CGO=0; C/CGO is required
only for race testing.

Core is **user-installed separately**: exact approved sing-box **1.14.3**
linux-amd64 digest. No core is bundled or automatically downloaded. Follow
[core verification](docs/core-versions.md); never replace an active core.

When an authorized release exists, obtain the archive/checksums from the same tag's
assets. Until then build from source. Linux archive installation:

```sh
sha256sum --check checksums.txt
# Inspect paths/types before extracting a downloaded archive.
tar -tzf tunnelctl_0.2.0-rc.1_linux_amd64.tar.gz
mkdir -m 700 ./tunnelctl-unpacked
tar -xzf tunnelctl_0.2.0-rc.1_linux_amd64.tar.gz -C ./tunnelctl-unpacked
mkdir -p "$HOME/.local/bin"
install -m 0755 ./tunnelctl-unpacked/tunnelctl "$HOME/.local/bin/tunnelctl"
"$HOME/.local/bin/tunnelctl" version --json
```

Use real Unix permissions: FAT/fuseblk drives are unsuitable for secrets, core or
detached executables. Installation only copies files; no connection, host network
change or boot persistence. Archives contain notices, dependency license texts,
offline docs and `SUPPORT.txt`.

macOS/Windows archives are **compile-only**, not supported client installations.
The following commands inspect artifacts, not native VPN functionality:

```sh
# macOS: compare this digest with the exact filename in checksums.txt.
shasum -a 256 tunnelctl_0.2.0-rc.1_darwin_arm64_compile-only.tar.gz
tar -tzf tunnelctl_0.2.0-rc.1_darwin_arm64_compile-only.tar.gz
```

```powershell
Get-FileHash .\tunnelctl_0.2.0-rc.1_windows_amd64_compile-only.zip -Algorithm SHA256
Expand-Archive -LiteralPath .\tunnelctl_0.2.0-rc.1_windows_amd64_compile-only.zip -DestinationPath .\tunnelctl-inspect
.\tunnelctl-inspect\tunnelctl.exe version --json
.\tunnelctl-inspect\tunnelctl.exe --help
# init/run are unsupported on Windows; do not deploy as a Windows VPN.
```

Checksums detect corruption, not independent authenticity. The draft workflow
requests GitHub build-provenance attestations. Once actually available, verify an
archive with `gh attestation verify ARCHIVE --repo mohammadhejazirad/tunnelctl`.
No local dry-run attestation/signing key is claimed.

## Quick start

Replace the core path with your protected, verified installation. Place the share
link in an owned `0600` regular file; never pass secret links in arguments.

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
opens Help. Profiles: `i` hidden link import, `f` private file, `/` search,
Enter details/select/config-check, `r` rename, `d` confirmed removal.
Subscriptions: `a` hidden add, `p/u` preview/update, `r` rename, `d` remove/retain.
Connect/Settings configure port, duration, authentication and an optional explicit
HTTPS probe; settings are menu-local. All menu starts are truly **detached**.
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
Unsupported feeds/protocols remain rejected; successful download is not connection.

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
are `0700`, secret files `0600`. Schema-2 `profiles.json` stores profiles and source
URLs atomically. This is permission-protected plaintext, not encryption at rest.
See [versioning/migration](docs/versioning.md).

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
go run ./cmd/release --tag v0.2.0-rc.1 --dry-run --out dist
TUNNELCTL_RELEASE_DIR="$PWD/dist" TUNNELCTL_RELEASE_TAG=v0.2.0-rc.1 go test ./test/release
```

Without explicit binary/core variables, real acceptance tests **skip**: not evidence
of connection success. See [tests](docs/testing.md), [manifest](compatibility.json),
[architecture](docs/architecture.md), [CLI](docs/cli.md),
[releasing](docs/releasing.md), [changelog](CHANGELOG.md) and
[execution checklist](docs/execution-checklist.md).

Pre-1.0 minor releases may change APIs with documented migrations; patches should
retain compatibility. App, output/storage schema and core versions are independent.
SemVer tags, including prereleases, drive release metadata. Workflow gates all
targets before preparing a **draft**, never an automatically public release.
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
