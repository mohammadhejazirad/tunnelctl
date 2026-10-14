# Actual test evidence — 2026-10-14

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
actionlint .github/workflows/quality.yml .github/workflows/release.yml
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

Remote GitHub Actions, native macOS/Windows/ARM64, PowerShell runtime, other Linux
distributions/musl, package managers, public-binary upgrade, multi-user hostile IPC,
every disk/power-loss boundary, privileged TUN/journal/watchdog/SSH/DNS/IPv6 rollback,
services, Xray, packet capture, performance/resource budgets and an independent
security/supply-chain review. Cross builds prove compilation only. The available
rootless network namespace and /dev/net/tun are not certification of host managers
or remote management recovery; Docker access is denied and no disposable host-like
VM acceptance environment was provisioned. Unsupported features remain disabled.
