# Actual test evidence — 2026-10-14

## Durable-storage increment — 2026-10-15

Public tests use synthetic fixtures and never read private samples. Store
subprocesses check SIGKILL at create/file-sync/rename/directory-sync boundaries,
RLIMIT_FSIZE write failure, 50 concurrent commits, XDG/CWD isolation, private
migration backups and corrupt/newer refusal. CLI/model tests cover persistent
settings and tags/favorites, draft discard, stale Save, permission failure,
idempotent init/reset and no auto-connect. Controlled real-core integration passed
in 14.241 seconds, including PTY Save -> CLI settings, stored defaults -> real proxy
request, active-session isolation and actual schema-2 executable upgrade/downgrade
refusal. Set TUNNELCTL_SCHEMA2_BIN for the optional old-executable test.
This is not physical reboot/power-loss or native non-Linux certification.

Initial complete race with the instrumented CLI and pinned core passed in
190.718 seconds (schema-2 reference enabled). Native archive acceptance passed
in 0.392 seconds for all five archives, including checksums/target metadata,
required storage guide and populated preferences/source/profile retention across
manual executable replacement/removal. The schema-1 reference was rebuilt from
pinned public commit a783dc63ff37359bc6e05359887c5272d723daf9; its actual upgrade
and old-decoder refusal passed separately with the final race client (11.612s).
One later gate initially selected a schema-2 executable for the schema-1 test;
that setup is invalid, not an application migration failure. The rebuilt schema-1
reference is used for the final complete gate. Native multi-UID users, a real full
filesystem, physical reboot/power loss and native macOS/Windows remain untested;
synthetic XDG roots, UID checks and permission/size-limited writes are separate evidence.

Final complete `go test -race -count=1 ./...` passed with the final race CLI,
verified schema-1 and schema-2 references, pinned core and synthetic development
harness (integration 208.275s). No historical upgrade test was skipped in that run.
The harness's samples were synthetic, not the root private file. Final five-target
persistence-delivery archives passed installation/checksum/metadata gates (0.382s).
Code formatting, vet, module verification, hygiene and actionlint also passed.

Final delivery: `dist/persistence-verified/`, local v0.2.0-rc.1 dry-run,
development=true, no publication. All five archives passed checksum, member,
target, version/storage-schema agreement and populated installation lifecycle gates
(0.361s); the release checks also passed race separately. The extracted final Linux
binary passed the complete controlled proxy/PTY/persistence/lifecycle suite with
both historical upgrade references and the synthetic harness (16.486s). These
post-package checks are recorded here after artifact creation; embedded offline
guides already contain the earlier successful candidate evidence and full recovery
instructions. No private input, snapshot, backup, session data or development tool
is included in those archives.

## Menu/private-harness increment

Public tests remain synthetic and independent of any real sample file. Added
model tests cover main-menu navigation, hidden/pasted imports, Unicode/control
sanitization, search, details/select/rename/remove, duration settings, real
controlled-TLS subscription add/preview/commit/refetch/remove and cancellation.
Exit waits for operation cancellation/cleanup before checking actual session state.
Redirected no-argument startup never reads stdin and returns help or valid JSON.

Linux PTY tests cover explicit `tui` and no-argument entry, connection settings,
real detached proxy start/request, leaving it running on exit, stale/empty
confirmed-session stop refusal and actual stop/cleanup. The development harness
is separately tested against a synthetic real Shadowsocks server and explicit
trusted-CA HTTPS target; config validation, readiness, TLS request, stop and cleanup
are separate assertions. Source/private-name/archive guard tests are synthetic.

Initial complete race with separately race-built CLI/harness passed (integration
155.607 seconds); ordinary real-core acceptance passed (12.963 seconds). Subsequent
review adds CLI config-check/explicit-probe equivalents and re-runs final gates.
Fresh CLI probe tests assert TLS refusal, trusted-CA success, credential/query
target refusal and no fabricated persisted connectivity status.
Final current-source ordinary core suite passed (13.885 seconds), second complete
real-core race passed (155.478 seconds), and rebuilt race CLI diagnostic acceptance
passed (12.839 seconds). Latest public full race and related harness/menu/subscription
race passed after final assertion review. The final five archive checks passed
(0.326 seconds), then the extracted final Linux binary passed real-core/PTY/TLS/
private-harness acceptance (14.212 seconds). See execution-checklist.md for the
current ignored dry-run directory and historical-vs-current evidence distinction.
After final bounded error/remedy rendering, dist/menu-delivery/ archives passed
membership/checksum/metadata checks (0.334 seconds) and their extracted Linux binary
passed all controlled integration tests (13.985 seconds); historical upgrade still
explicitly skipped. Related UI/CLI race and final vet/format/hygiene checks passed.
Historical baseline upgrade tests skip in this increment unless
TUNNELCTL_PREVIOUS_BIN is supplied; prior evidence below is not a new upgrade run.
Private endpoint reports are never checked into this document or release assets.

For harness acceptance, additionally build `cmd/private-test` and set
`TUNNELCTL_PRIVATE_TEST_BIN` to its protected executable. The CI race job does
this with `-race`, and packaged acceptance builds the development tool separately
(it is not distributed). See [private activation](private-testing.md) and
[menu guide](tui.md). No native non-Linux/TUN acceptance is implied.

## Earlier baseline evidence

Local environment: Ubuntu 26.04.1 LTS amd64, kernel 7.0.0-38-generic, UID 1000.
Go 1.27.2 and approved sing-box 1.14.3 match recorded digests. Source is fuseblk
with 0777/nobody semantics; secrets/core/detached executables are staged privately
on Unix-permission-supporting /tmp, not this source drive.

The absent C compiler was resolved **without root or host package installation**:
isolated gcc-15-x86-64-linux-gnu and libgcc-15-dev 15.2.0-16ubuntu1 packages were
downloaded, checked against apt SHA512 metadata and extracted under private /tmp.
Existing binutils, C preprocessor and libc development components were used.
Race CGO is separate from CGO=0 ordinary/release builds.

Passed locally:

- Unit tests, go vet, gofmt checks, go mod verify and git diff whitespace check.
- Full race suite with a **separately race-built CLI** and real pinned core. An
  expanded full run including actual baseline upgrade completed in 142.950 seconds
  for integration; no race reported. Private child report capture also checks races
  hidden by production detached diagnostics suppression.
- Real trusted loopback HTTPS subscription tests: preview without mutation,
  ETag/Last-Modified/304, stable IDs/labels/selection, malformed/empty/500 rejection,
  TLS refusal, bounded gzip/decode/count, SSRF, mixed DNS answers, reserved address
  policy, cross-origin/downgrade/redirect-limit refusal and cancellation.
- Concurrent source lock, staged rename/removal CAS conflicts, recovery after
  cancellation and **actual filesystem write-permission commit failure** preserve
  healthy data/concurrent changes. These are not power-loss/disk-full certification.
- CLI subscription add -> preview -> commit -> selection -> real proxy request;
  corrupted refresh preserves private data and source removal leaves session alive.
- Real Shadowsocks core requests for all three approved AEAD methods, authenticated
  HTTP/SOCKS, hostname delivery/resolution and TLS-verified CONNECT with a trusted
  synthetic CA. Wrong credentials fail without reaching the target.
- Port/session collision, duration, repeated stop, SIGINT/SIGTERM/SIGHUP, core crash,
  supervisor SIGKILL/parent-death termination and stale cleanup.
- Detached real request and isolated exec/curl, child exit 23 preserved, unrelated
  detached session identity unchanged, state logs and repeated stop.
- TUI model compact/resize/control-sequence checks and Linux PTY smoke: render,
  keyboard navigation/help, quit, no synthetic password disclosure. This is not
  comprehensive visual/accessibility certification or native macOS/Windows TUI.
- Schema-1 migration backup, failed-write preservation and newer schema refusal;
  strict duplicate/unknown/trailing/deep/invalid-UTF8 JSON and permission checks.
- Actual executable upgrade from baseline commit a783dc6 (0.1.0 development),
  profile identity/selection and exact schema-1 backup preserved; old binary refuses
  newer data without changing bytes; upgraded proxy sends a real request. Dedicated
  race test passed in 10.61 seconds; included in the expanded full run as well.
- Ordinary/set-ID executable checks; proxy refuses set-ID/file-capability elevation.
- SemVer/prerelease/build-metadata validation, dirty marker and tag mismatch refusal.
- Local release builder produced five archives, SBOM, checksums and license texts;
  initial extraction/installed help/version/init/status/stop/manual replacement/
  uninstall tests passed. Final archive/extracted-core evidence is recorded in the
  execution checklist; artifacts are local dry-runs, not a public release.
  Extracted final-binary real-core suite passed in 7.968 seconds.
- govulncheck **v1.9.0**: `No vulnerabilities found.` This is time-specific
  known-vulnerability assessment, not absence of defects.
- actionlint **v1.7.12** validated both workflow files. Narrow repository hygiene
  scanner passed; it is not exhaustive secret detection.

Baseline parser fuzz evidence: 22,541 executions in a 10-second bounded campaign,
approximately 11.026 seconds including shutdown. Repeat for code changes; a bounded
fuzz run does not prove absence of bugs. Test lifecycle fakes only cover rejection,
readiness timeout, cleanup conflict and stop escalation, not actual connectivity.

Real-core acceptance explicitly skips without both test variables. Artifact tests
skip without release directory/tag. Never count these skips as network/package
success. Loopback tests use approved execution outside the sandbox's socket
restriction. No host default-route/DNS/firewall/global-proxy/service changes.

## Reproduce

Use Go 1.27.2, approved core, ordinary Linux user, permitted loopback and private
local storage. A supported C compiler is required only for race:

```sh
go mod download
go mod verify
go vet ./...
go test -count=1 ./...
make integration CORE=/absolute/path/to/approved/sing-box
make race CORE=/absolute/path/to/approved/sing-box
go test ./internal/importer -fuzz=FuzzParse -fuzztime=10s -parallel=2
go run ./cmd/audit
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
govulncheck ./...
go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
actionlint .github/workflows/*.yml
go run ./cmd/release --tag v0.2.0-rc.1 --dry-run --out dist
TUNNELCTL_RELEASE_DIR="$PWD/dist" TUNNELCTL_RELEASE_TAG=v0.2.0-rc.1 go test -v ./test/release
```

Artifact tests inspect tar/ZIP membership, licenses/docs, digests, target metadata,
CGO=0 and injected tag/version on **all** binaries; native execution occurs only
for Linux amd64. Extract that Linux binary into a protected directory, then set
TUNNELCTL_TEST_BIN to it and TUNNELCTL_TEST_CORE to the approved core and execute
`go test -count=1 -v ./test/integration` for real packaged lifecycle evidence.

For historical upgrade tests, build cmd/tunnelctl from commit
a783dc63ff37359bc6e05359887c5272d723daf9 in a new private temporary source directory,
then set TUNNELCTL_PREVIOUS_BIN to that protected executable. CI uses git archive
and Go -C without altering the worktree or creating tags.

## Not established

Remote GitHub Actions, native macOS/Windows/ARM64, PowerShell runtime, additional
Linux/musl core networking combinations, public-installer upgrade, multi-user hostile IPC,
every disk/power-loss boundary, privileged TUN/journal/watchdog/SSH/DNS/IPv6 rollback,
services, Xray, packet capture, performance/resource budgets and an independent
security/supply-chain review. Cross builds prove compilation only. The available
rootless network namespace and /dev/net/tun are not certification of host managers
or remote management recovery. No disposable host-like VM acceptance environment
was provisioned. Unsupported features remain disabled. See the later packaging
evidence below for newly authorized offline Docker package tests; those do not
supersede the TUN/host-network exclusions.

## Packaging increment (2026-10-15)

GoReleaser check and real CGO=0 five-target builds passed; nFPM generated deb,
rpm, Arch and apk for amd64/arm64, plus two PKGBUILDs. Hash-pinned appimagetool
generated both AppImage architectures. Archive tests actually extracted/executed
Linux amd64 and verified all five targets' build/version metadata, documents,
module licenses, privacy exclusions and stored-data preservation. New tests cover
numeric native version order/limits, exact clean-tag binding in a disposable Git
repository, missing/extra/tampered asset rejection and private/symlink rejection.

Actual offline amd64 package collision/install/remove/help/version tests passed
on Ubuntu 24.04.5, Debian 13.7, Fedora 43, Arch 20261011.0.609700 and Alpine
3.22.6. Arch makepkg parsed the actual PKGBUILD as UID 1000, then built it as an
ordinary builder user; the resulting package passed install/help/version/remove.
Final repeated container gates also checked root ownership, 0755 directories and
0644 documents. Minimal Docker images exclude docs globally; only the disposable
test invocation overrides those exclusions to inspect this package's full content.
No host packages or
network settings changed. Fedora's /usr/sbin symlink alias and apk's non-aborting
preinstall failure behavior were found and addressed, not suppressed.

AppImage transient extraction metadata/stdin/JSON/redaction/persistence/exit tests
passed. Directed-signal/core-crash tests exposed upstream's extra parent process;
run/start/exec now explicitly refuse transient sessions. Stable extracted payload
passed actual controlled-core requests, duration, all three stop signals,
supervisor/core crash and detached/exec. /dev/fuse is absent: mounted execution
is not claimed tested. Private sample file was never opened; all fixtures synthetic.

Local finalize deliberately failed on missing native macOS pkg, proving the
incomplete matrix cannot get checksums/publication as a full release. Windows MSI,
macOS pkg and ARM native lifecycle, signing/notarization, full GitHub dry-run and
live GitHub upload/download/promotion are **not executed locally**. No public tag
or release was created. Native gates are mandatory when the workflow runs.

Remote Packaging validation on 2026-10-14 passed its base build, Linux amd64/arm64
package checks and native macOS amd64/arm64 pkg lifecycle jobs. Windows stopped
before installation with WiX WIX0367 because one auto-GUID component owned six
unversioned files. The manifest now assigns one file per auto-GUID component and a
Go regression test enforces that rule; a remote Windows rerun and aggregate remain
required before release evidence is complete.

After the application guards and packaging tests: `go test ./...`, `go vet ./...`,
`go mod verify`, hygiene and actionlint passed. Full `go test -race -count=1 ./...`
passed with GCC 15/CGO=1, real race-instrumented CLI/private harness and historical
schema-1/schema-2 executable upgrade tests (integration package 208.589 s).
Ordinary tests without explicit core/binary still skip opt-in network acceptance.

Final public tests and a second public race pass covered the final packaging
changes. Five-target archives passed again (0.352s); the rebuilt AppImage's
metadata checks passed (0.186s), and its stable extracted payload passed the
controlled real-core lifecycle suite (5.184s). The actual publisher script also
passed synthetic transport tests for stable/prerelease promotion flags, existing
release refusal, API failure, upload failure, tampered download and manual-run
refusal. These tests verify control flow and real local checksum validation, not
live GitHub authentication or API behavior.

The final deb payload was separately extracted into private Unix storage and
passed the core-enabled integration suite (13.966s). Optional old-executable and
development-harness checks were not enabled in this last payload-only run; they
were included in the full race run recorded above.

Local candidate output is `dist/packages-delivery/` (dirty development dry-run).
It is not a complete public release: missing native installers are not fabricated
and the full-set finalization gate refuses them. All local packaging fixtures are
synthetic; the root private sample file was never read.
