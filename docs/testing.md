# Test evidence — 2026-10-14

Environment: Ubuntu 26.04.1 LTS, Linux 7.0.0-38-generic x86_64, UID 1000.
Go 1.27.2 and the exact official sing-box 1.14.3 linux-amd64 binary were verified
against recorded SHA-256 digests. Tools/caches were created in a private /tmp
directory because Go/core were not initially available on PATH. The approved core
and license are at `/tmp/tunnelctl-tools.SSdBLI/sing-box-1.14.3-linux-amd64/`.
The app binary is `bin/tunnelctl`. A trial copy into the project drive was rejected:
that fuseblk mount exposes 0777 modes and nobody ownership despite install -m.
The redundant trial copy was removed; private runtime/profile state and the core
remain on permission-supporting Linux filesystems. This verifies refusal rather
than weakening permissions to accommodate the project mount.

Executed successfully:

- `go build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl`
- `go test -count=1 -v ./...` with TUNNELCTL_TEST_BIN and TUNNELCTL_TEST_CORE set
- `go vet ./...`
- `go test ./internal/importer -fuzz=FuzzParse -fuzztime=10s -parallel=2`
- Cross builds with CGO_ENABLED=0: Windows amd64, macOS amd64/arm64, Linux arm64

Real-core suite passed in approximately four seconds. Each synthetic server binds
127.0.0.1, uses a supported AEAD method and a synthetic password, and forwards to
controlled loopback HTTP/TLS targets. Requests actually traverse the generated
mixed listener and Shadowsocks outbound; a wrong upstream password produces failure
and never reaches the target. Local listener authentication is required in the
method matrix: unauthenticated requests fail, authenticated requests succeed.
SOCKS5 proxy-side hostname delivery and upstream hostname resolution were tested.
HTTPS uses HTTP CONNECT and an explicitly trusted test certificate with verification
enabled, not InsecureSkipVerify. IPv6 literal configs passed real `check`; IPv6
traffic behavior was not tested or certified.

Fault assertions passed: unavailable port, session lock collision, repeated stop,
duration deadline, SIGINT/SIGTERM/SIGHUP, core SIGKILL, supervisor SIGKILL and core
parent-death termination, stale metadata detection and safe cleanup. A test-owned
helper process ignores SIGTERM to prove graceful-timeout escalation to forced kill.
Lifecycle fakes exercise rejected config, readiness deadline and an unexpected
runtime file type that must produce code 7 and remain intact. These fakes do not
replace real networking evidence.

Unit tests cover URI encoding/Base64 variants, endpoint/cipher/field validation,
input/decode bounds, terminal sequence removal, credential redaction, valid CLI
JSON, private file import and permissions, same-filesystem atomic transactions,
locking, symlink refusal, unsupported schemas, duplicate JSON fields, excessive
nesting and trailing values. Parser fuzzing found no failing input in the recorded
bounded campaigns; that does not establish absence of defects.
The final 10-second fuzz campaign completed 22,541 executions (11.026 seconds wall
time including shutdown); earlier campaigns exercised additional inputs.

The sandbox initially forbade socket creation. Real loopback tests were run with
approved execution outside that socket restriction. No test changes default
routes, system DNS, firewall, global proxy, shell files or boot services. No root,
remote production server or public VPN server was used.

Race verification did **not run**: default `go test -race` reported that CGO was
required; explicitly enabling CGO then failed because gcc was unavailable. Native
macOS/Windows/arm64, PowerShell, the broader Linux matrix, multiple-user hostile IPC,
disk-full/power-loss fault injection, packet captures, DNS/IPv6 leak policies and
performance targets remain unverified. Cross builds establish compilation only.

## Reproduce

Use the pinned Go/core, an ordinary Linux user and a private local filesystem.
Allow local sockets. From the repository:

```sh
go build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl
TUNNELCTL_TEST_BIN="$PWD/bin/tunnelctl" \
TUNNELCTL_TEST_CORE="/absolute/path/to/approved/sing-box" \
go test -count=1 -v ./...
go vet ./...
go test ./internal/importer -fuzz=FuzzParse -fuzztime=10s -parallel=2
```

`make integration CORE=/absolute/path/to/approved/sing-box` builds and runs the
same unit/real-core suite. Without the two explicit test environment variables,
real acceptance tests skip with an explanation. After installing a C compiler,
run `CGO_ENABLED=1 go test -race ./...`, then run the integration suite with a
race-instrumented application binary as well; race instrumentation on the test
runner alone does not instrument a separately built CLI child.
