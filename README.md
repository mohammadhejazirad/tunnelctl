# tunnelctl

A Go terminal-only proxy client. Phase 0 and Phase 1 implement a real foreground
CLI on Linux amd64, using the external **sing-box 1.14.3** core. No desktop,
browser, dashboard, privileged networking or boot service is involved.

Supported profiles: **Shadowsocks SIP002**, TCP, `aes-128-gcm`, `aes-256-gcm`,
`chacha20-ietf-poly1305`, without plugins. Other protocols, subscriptions, raw
configs, UDP, TUN, detached sessions and TUI are not implemented. Import success
means parsing and storage succeeded; it does not prove upstream connectivity.

## Build

Install Go **1.27.2**, then:

```sh
go mod download
go build -buildvcs=false -trimpath -o bin/tunnelctl ./cmd/tunnelctl
./bin/tunnelctl --help
```

`make build` also works. See [the compatibility manifest](compatibility.json)
and [core verification instructions](docs/core-versions.md) for the exact core
archive and hashes. Install it manually, without root, and pass its absolute
path using `--core`. The application performs no automatic downloads or updates.
Only the approved linux-amd64 binary hash is accepted; matching a version string
alone is insufficient. An existing different sing-box installation may be refused.

In this development environment the approved core and license are available in
the private temporary tools directory. You can use:

```sh
./bin/tunnelctl --core /tmp/tunnelctl-tools.SSdBLI/sing-box-1.14.3-linux-amd64/sing-box doctor --json
```

This path is temporary and must be replaced with a verified installation for
long-term use. The current project drive is fuseblk and exposes files as 0777,
owned by nobody. Do not store secrets or the approved core on this drive; the
application intentionally refuses them. Keep private data/core on a filesystem
supporting real Unix permissions, such as the user's Linux home or private /tmp.

## Use

Run as an ordinary user. Place your actual share link in a private `0600` regular
file, or supply it through stdin. Do not put credentials in command arguments.
Replace the core path below with your verified executable's absolute path.

```sh
./bin/tunnelctl init
./bin/tunnelctl profile import --file /absolute/path/to/private-profile.txt
# Alternatively: ./bin/tunnelctl profile import --stdin < /private/profile.txt
./bin/tunnelctl profile list --json
./bin/tunnelctl profile show PROFILE_ID --redacted
./bin/tunnelctl profile select PROFILE_ID
./bin/tunnelctl --core /absolute/path/to/sing-box doctor --json
./bin/tunnelctl --core /absolute/path/to/sing-box run --mode proxy --duration 10m
```

`run` stays in the foreground, listening on **127.0.0.1:2080**. `--profile`
overrides the selected profile; `--listen-port` chooses another unprivileged port.
In a second terminal, using the same user and storage:

```sh
./bin/tunnelctl status --json
./bin/tunnelctl stop
./bin/tunnelctl stop  # Safe to repeat
```

Ctrl+C, SIGTERM and SIGHUP also stop the foreground session. A duration begins
after listener readiness; shutdown may take up to six additional seconds.
Terminal hangup is handled when SIGHUP is delivered. Abrupt supervisor death
triggers Linux parent-death termination of the owned core; next `stop` or `run`
reconciles stale runtime metadata. See [offline recovery](docs/recovery.md).

The proxy does not configure other applications. An explicit example request
below sends traffic to **example.com**, only when you run it:

```sh
curl --proxy http://127.0.0.1:2080 https://example.com/
curl --proxy socks5h://127.0.0.1:2080 https://example.com/
```

Loopback is reachable by other local users. On a shared machine, use
`--proxy-auth-file /private/listener-auth.json`; its format is shown in
[the synthetic example](examples/proxy-auth.json). The file must be `0600` and
owned by you. HTTP/SOCKS clients must then authenticate. Avoid exposing those
credentials in their command arguments; use a client-supported private file.

Storage uses separate XDG config/state/cache/runtime directories. For an isolated
trial, pass `--data-dir /private/trial` consistently to every command. Existing
directories must be owned by you with `0700` permissions, with no symlink
components. Secret files are plaintext protected by Unix permissions, not encrypted.

## Verification and current limits

```sh
go test ./...
go vet ./...
make integration CORE=/absolute/path/to/approved/sing-box
go test ./internal/importer -fuzz=FuzzParse -fuzztime=10s -parallel=2
```

The real-core tests create synthetic Shadowsocks servers and HTTP/TLS targets on
loopback. They verify real requests, listener authentication, wrong credentials,
port conflicts, concurrent sessions, duration, signals, core crashes and supervisor
death. Tests needing a core explicitly skip if the test environment variables
are absent. Mock lifecycle tests cover rejection, readiness timeout and cleanup
conflicts; they are not evidence of network connectivity.

See [recorded test evidence](docs/testing.md). Linux amd64 was tested in this
environment. Other binaries may compile without providing native runtime support.
No throughput, DNS-leak, kill-switch or general VPN security claim is made.
`connection_tested` stays false: runtime status does not run unsolicited probes.
Statistics are reported as unavailable. Core logs are disabled/suppressed because
they can expose secrets; automatic upstream failure classification is deferred.

Architecture, security, CLI, compatibility, recovery and roadmap details are in
[docs/architecture.md](docs/architecture.md), [docs/security.md](docs/security.md),
[docs/cli.md](docs/cli.md), [docs/compatibility.md](docs/compatibility.md),
[docs/recovery.md](docs/recovery.md) and [docs/roadmap.md](docs/roadmap.md).
