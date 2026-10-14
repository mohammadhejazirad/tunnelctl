# Terminal VPN — Architecture and Agent Build Specification

Version: 1.0 | Research date: 2026-10-14 | Language: English

Status: proposed product specification, not an implemented or audited VPN.
Working command name: `tunnelctl` (check naming conflicts before publishing).
MUST means a release requirement; SHOULD permits a documented exception.

## 1. Decision

Build a compiled Go application, with Cobra commands and an optional Bubble Tea / Lip Gloss terminal UI. Use sing-box as the first external networking core. Introduce an Xray adapter after the initial Linux safety gates pass. Run one core per session by default. Do not implement encryption, transport protocols, or a packet stack from scratch.

This is a terminal-native network application, not a collection of shell scripts. Bash and PowerShell may be thin installation/completion wrappers; neither is the main implementation language. CLI commands must remain fully usable without a TUI, mouse, browser, desktop session, or systemd.

The target is a client running on a desktop or server and connecting to an existing remote proxy server. Hosting a public VPN server, multi-user gateway, forwarding other machines' traffic, router firmware, and mobile applications are out of scope initially.

### Language trade-offs (engineering judgment)

| Choice | Strength | Cost for this project | Decision |
|---|---|---|---|
| Go | Native executables, concurrency, straightforward process supervision, strong CLI/TUI ecosystem | OS-specific network code still required; cross-compilation does not validate behavior | Recommended |
| Rust + clap + Ratatui | Strong resource control and native distribution | Higher implementation burden if the team lacks Rust experience | Good alternative for an experienced Rust team |
| Python + Typer + Textual | Fast prototyping and expressive TUI | Runtime/bundling and privileged deployment add operational work | Prototype or auxiliary tooling |
| TypeScript + Ink | Familiar to web teams | Node runtime/bundling adds complexity without improving network control | Not preferred |
| Bash / PowerShell | Useful glue around existing cores | Harder portable parsing, state machines, testing and recovery | Wrappers only |

Go does not guarantee a static, dependency-free binary for every dependency or build configuration. Audit CGO and OS dependencies before publishing artifacts.

## 2. Core selection

| Core | Proposed role | Selection rule |
|---|---|---|
| sing-box | Default for proxy and TUN | Use only combinations validated against the pinned release |
| Xray-core | Optional second backend | Use for verified protocol/transport capabilities or explicit user selection |
| Other cores | Deferred | Add only for a documented unmet need and an independent adapter test suite |

Current official Xray documentation includes a TUN inbound for Windows, Linux, macOS and FreeBSD. Do not repeat the obsolete assumption that Xray has no native TUN. Neither core is universally fastest or most compatible: measure the required protocol, transport, machine and network. Choosing sing-box first is a scope and integration decision, not a benchmark claim. [S1–S4]

Do not chain Xray and sing-box by default. Chaining introduces DNS ownership, UDP forwarding, loop prevention, metrics and recovery complexity. A future chain must be an explicit experimental feature with its own tests.

Keep cores as supervised executable processes rather than importing their internals in the first release. This isolates upgrades and schemas, but does not remove security or distribution-license responsibilities. Track licenses and notices from the exact artifacts shipped; perform a distribution review before release.

### Version contract

Commit a compatibility manifest containing application version, Go toolchain, module versions, core release/tag, artifact SHA-256, source URL, target OS/architecture, build tags, required runtime components and tested features. Never select `latest` at runtime. Documentation may describe unreleased or newer features: consult documentation/source matching the pinned tag. Check core configuration before any network mutation. Refuse an untested major/schema change with an actionable error.

Capability checks must cover protocol × transport × TLS/security × mode × OS × core version, not protocol names alone. Never discard unknown critical fields to make an import appear successful. A rejected profile must explain which field is unsupported and by which backend.

## 3. User journeys and operating modes

1. Import a single share link from a private file or hidden prompt, select the profile, and run a local proxy for ten minutes.
2. Add an HTTPS subscription, preview parsed profiles, update it atomically, search and select a profile.
3. Run an application with proxy environment variables without editing shell startup files.
4. On Linux, inspect a TUN network plan, activate a protected session, and restore owned changes on stop.
5. Inspect connection state and redacted diagnostics over SSH with ordinary commands.
6. Explicitly install an optional background service after foreground behavior is validated.

### Proxy mode — default

Expose HTTP and SOCKS5 on loopback only, using a supported mixed listener or separate ports. Do not modify global routes, firewall, DNS, system proxy, shell configuration or browser settings. An application uses the proxy only if it supports and is configured for it. A loopback proxy is not automatically private from other local users; support listener authentication and document the multi-user boundary.

`exec` sets proxy variables only for its child. Programs may ignore them. `socks5h` can request proxy-side hostname resolution in compatible clients; it is not a universal DNS-leak guarantee. UDP and ICMP are separate capabilities and must not be implied by HTTP proxy support. Public/LAN listeners are deferred initially.

### TUN mode — explicit

Capture selected host traffic through a virtual interface, with OS-specific route, DNS and firewall handling. Require a supported privilege mechanism and preflight checks. Do not claim that every process or protocol is necessarily captured. IPv4, IPv6, DNS, bypass traffic and unsupported protocols require explicit policy and tests.

### Session ownership

- `run`: foreground session; Ctrl+C requests graceful stop and cleanup. Terminal loss normally stops the session. A duration deadline belongs to the session supervisor, not the TUI.
- `start`: explicit detached supervisor session, independent of terminal lifetime.
- `tui`: attaches to a running session or manages profiles. Quitting the TUI does not stop a detached session; show this clearly.
- `stop`: idempotent stop, followed by owned-resource reconciliation.
- `service install`: explicit persistence; never enabled as a side effect of `run`, import or installation.

## 4. Command contract

The following are proposed application commands, not existing executable commands.

```text
tunnelctl init
tunnelctl tui
tunnelctl profile import --file ./profile.txt
tunnelctl profile import --stdin
tunnelctl profile list --json
tunnelctl profile show PROFILE --redacted
tunnelctl profile select PROFILE
tunnelctl profile remove PROFILE
tunnelctl subscription add --name personal --url-stdin
tunnelctl subscription list
tunnelctl subscription update personal --preview
tunnelctl subscription update personal
tunnelctl subscription remove personal
tunnelctl run --profile PROFILE --mode proxy --duration 10m
tunnelctl start --profile PROFILE --mode proxy
tunnelctl exec --profile PROFILE -- curl https://example.com
tunnelctl plan --profile PROFILE --mode tun
tunnelctl run --profile PROFILE --mode tun --management-cidr CIDR
tunnelctl confirm SESSION
tunnelctl status --json
tunnelctl stop
tunnelctl logs --follow
tunnelctl doctor
tunnelctl recover --dry-run
tunnelctl recover --session SESSION
tunnelctl core list
tunnelctl core install sing-box --version APPROVED_VERSION
tunnelctl core verify
tunnelctl core update --check
tunnelctl service install
tunnelctl service uninstall
tunnelctl completion bash|zsh|fish|powershell
tunnelctl version --json
```

Use hidden interactive input for secrets when a terminal is available. Avoid tokens in command arguments, shell history, process listings and diagnostics. A `--stdin` parser and private file workflow are mandatory. Do not recommend pasting secret-bearing links directly into shell commands.

Global flags: `--config`, `--json`, `--plain`, `--no-color`, `--non-interactive`, `--timeout`, `--verbose`. Respect `NO_COLOR`, `TERM=dumb`, terminal resize and redirected output. No prompts or spinners in noninteractive/JSON mode. Write machine-readable results to stdout and diagnostics to stderr. `--verbose` must not reveal credentials.

Stable exit codes: 0 success; 1 unexpected failure; 2 invalid input; 3 unsupported capability; 4 permission required; 5 network/upstream failure; 6 state conflict; 7 cleanup incomplete. JSON includes `schema_version`, `ok`, `data`, and a sanitized error with `code`, `message`, `remediation`. A failed command never returns success merely because partial work completed.

`exec` must preserve the child's exit status after successful cleanup and document how cleanup failures are reported. It starts an isolated owned session or reuses a compatible session explicitly; it must never stop another user's session.

## 5. Terminal UX

All authored UI, help, errors and documentation are English. Preserve user-supplied Unicode profile names while sanitizing terminal control characters.

Screens: Overview, Profiles, Subscriptions, Routing, Sessions, Logs, Diagnostics and Settings. Overview displays mode, active profile, backend/version, uptime, deadline, traffic counters, local listener, DNS/IPv6 policy, bypasses and cleanup status. Do not invent counters when a backend does not supply them; display `Unavailable`.

Use restrained color, clear spacing, visible keyboard hints and text labels for every status. No required Nerd Fonts, emoji, mouse, animation, or truecolor. Support a compact 80×24 layout and smaller terminals with scrolling. Provide arrow keys and optional Vim-style navigation, `/` search, `Enter` select, `Esc` back, `?` help. Do not bind an easy accidental keystroke to destructive actions.

Use Bubble Tea for state/events, Bubbles where useful, and Lip Gloss for styling; keep compatible major versions pinned. Prefer a separate rendering model from domain/session state. TUI network requests must not block the update loop. Refresh at a modest rate, pause hidden views, and keep a bounded log buffer. [S5–S7]

## 6. Architecture

```text
cmd/tunnelctl/             entry point only
internal/cli/              Cobra command definitions and output contracts
internal/tui/              terminal views and view models
internal/app/              shared use cases, authorization and orchestration
internal/domain/           normalized profile, policy and session models
internal/importer/         bounded URI/subscription parsers
internal/subscription/     fetch, validation, diff and transactional update
internal/core/             backend interfaces and capability registry
internal/core/singbox/     versioned renderer and process adapter
internal/core/xray/        optional second adapter
internal/session/         supervisor, deadlines and process identity
internal/netplan/         mutation planning and recovery journal
internal/platform/        Linux, Darwin and Windows implementations
internal/privilege/       narrow authenticated privileged operations
internal/store/           private atomic persistence and migrations
internal/diagnostics/     doctor, redaction and local support bundle
test/integration/          disposable VM and namespace scenarios
docs/                     user guide, architecture, threat model, recovery
```

Both CLI and TUI call the same application services. The TUI never directly mutates networking. Prefer explicit Go configuration structs and a small dependency graph over a framework-heavy service locator.

Core adapter responsibilities: probe version, report capabilities, render config, validate config, launch, determine readiness, report health/metrics, stop and reconcile owned resources. Launch with an absolute verified executable path and argument arrays, never shell-concatenated input. Handle PID reuse using start time/process identity; never kill by name or a stale PID alone. Use process groups on Unix and an appropriate Windows job/process lifecycle strategy.

One supervisor owns each session. Serialize host-wide TUN mutations with a machine-scoped lock; per-user locks alone are insufficient. Use private Unix sockets with peer credential verification, and Windows named pipes with restrictive ACLs and identity verification. Do not expose a remote management TCP API by default.

## 7. Network safety — release blocking

Zero risk cannot be guaranteed. The objective is limited changes, clear ownership, verified cleanup and documented recovery. No fault-handling strategy can execute during power loss; recover durable owned state on the next start and provide manual instructions.

### Privilege boundary

Run parsing, downloads and UI unprivileged. Proxy mode requires no administrative privileges on ordinary unprivileged ports. TUN uses a narrowly scoped privileged supervisor/helper. Authenticate the caller, validate every request, reject arbitrary executable paths/configs and shell commands, and prevent symlink/TOCTOU substitutions. Executables/config files used by a root helper must not be replaceable by an unprivileged process between verification and execution. Do not grant persistent capabilities to a user-writable core binary. Minimize elevated lifetime without assuming every core supports dropping privilege safely.

### Transaction and ownership

State machine: `IDLE → PREFLIGHT → PREPARED → STARTING → ACTIVE → STOPPING → IDLE`; failures enter `ROLLING_BACK` and, when needed, `RECOVERY_REQUIRED`.

1. Acquire locks; refuse ambiguous existing state.
2. Inspect interfaces, routes/rules, DNS owner, firewall, SSH context, other VPNs, ports and kernel features.
3. Build a deterministic plan listing each intended resource, prior value, expected owned value and inverse operation.
4. Validate generated core configuration without changing host networking.
5. Persist and fsync a write-ahead journal before each mutation. Record applied/uncertain states durably.
6. Start an independent recovery watchdog before activating routes. Ensure it survives UI death and the relevant supervisor/core failures; document its failure domain.
7. Apply changes, verify listener readiness, DNS, routing and a configurable real proxy request. Process existence alone is not readiness.
8. Commit only after verification and any remote-management confirmation.
9. On stop/failure, stop traffic safely, undo owned effects in policy-aware order, then verify and report remaining differences.

Use dedicated table/rule allocations, interface identifiers, firewall namespaces and session tags; collision-check before mutation. Never flush all nftables/iptables rules, reset a firewall, blindly replace the default route, kill unrelated core processes or overwrite the whole DNS configuration. Restore only when the current value still matches the owned value; surface conflicts caused by NetworkManager, DHCP, another VPN or an administrator.

Choose one route owner per backend/platform. If the core owns auto-routing, the adapter must model the exact pinned version's side effects and reconcile leftovers; the app must not install competing duplicate rules. If resources cannot be reliably enumerated and reversed, that configuration is unsupported. Journal snapshots are evidence, not authorization to replay the entire old network state.

### Linux specifics

Check `/dev/net/tun`, effective capabilities, kernel features, policy routing, firewall support and DNS manager. Feature-detect rather than branching only on distribution name. sing-box documents Linux `auto_redirect` as an nftables-based option used with `auto_route`; enable only with verified support and tested resource ownership. Do not treat one option as a universal safety guarantee. [S3]

Prefer supported per-interface resolver integration. Never replace `/etc/resolv.conf` blindly or break a managed symlink. For an unknown DNS manager, refuse the unvalidated host-wide DNS change and explain supported alternatives. Do not disable SELinux, AppArmor, firewalld or UFW to make the program work. Containers, restricted VPS instances and WSL require separate capability checks; they are not equivalent to a normal Linux host.

### SSH and remote servers

Remote TUN is high risk. Detect SSH environment when present, but do not trust it as the sole source of management routes. Support explicit management CIDRs and inspect the actual gateway/interface and policy rules. Exempt the upstream proxy endpoint and approved management destinations to prevent recursion and lockout; account for IPv4 and IPv6. Never broadly exempt all public traffic.

Before network activation, arm a short rollback deadline (proposed 30 seconds). A confirmation must arrive through a post-change management connection before the deadline; a previously running local command cannot establish external reachability. Document opening a second SSH connection and running `confirm SESSION`. Automated deployments require an equivalent external probe/confirmation. If protection cannot be implemented or verified, refuse remote TUN and keep proxy mode available. Existing forwarded/production server traffic is outside the initial TUN support scope.

### DNS, IPv6 and failure policy

Offer explicit IPv6 policies: tunnel, block for the session, or intentional bypass. Never silently bypass IPv6 while advertising full-tunnel protection. Handle upstream endpoint DNS bootstrap without routing loops; disclose any intentional bootstrap DNS traffic. Test TCP/UDP DNS, application DoH behavior, split DNS, local resolvers and reconnects. Fake-IP is deferred until explicit tests cover it.

Define `on-failure=restore` as the default: remove owned changes and return normal connectivity; this is fail-open and must be visible. A later `on-failure=block` policy installs tested leak-blocking rules before routing and preserves them on a core failure. Recovery must know whether to reconnect, retain blocking or explicitly unblock. A strict route option alone is not a complete kill switch. Block mode needs an emergency local recovery procedure and cannot be shipped as a label without packet-level tests.

## 8. Import and subscription specification

Initial candidates: VLESS, VMess, Trojan and Shadowsocks share links; HTTP/SOCKS endpoints where safely supported. Treat Hysteria2, TUIC and other formats as later capabilities, not promises of universal parsing. Define fixtures for each accepted dialect, including IPv6 literals, percent encoding, URL-safe/base64 padding, duplicate query keys, missing fields and transport/security options.

Accept single links and newline-separated lists; support explicitly recognized base64 subscription envelopes. Detect unsupported JSON/YAML formats and explain the limitation. Full sing-box/Xray/Clash configurations are NOT ordinary subscriptions: they can contain listeners, routes, files and privileged behaviors. Defer raw-config import; never execute imported configuration directly.

Subscription input is untrusted data. Default limits: 5 MiB compressed and 10 MiB expanded, 10,000 profiles, 64 KiB per link, 3 redirects, 20-second total deadline. Reject parser bombs, duplicate structural fields, invalid ports and control sequences. Escape OSC/ANSI content before display, including hyperlinks and clipboard-control sequences.

Use HTTPS and certificate validation. Reject downgrade redirects and cross-origin credential forwarding. Block loopback, private, link-local and cloud metadata destinations by default at resolution and actual connection time, including redirects and IPv6 equivalents. Offer a narrow explicit private-subscription option for legitimate LAN sources. Prevent DNS rebinding; do not assume a string hostname check is enough. Fetch unprivileged. No automatic third-party subscription converters: they would receive secrets.

Store a subscription's protected URL separately from its display name. Deduplicate by canonical connection identity, not label. Preserve user labels and active selections. Stage fetch/parse/validate/diff and commit atomically. Malformed or unexpectedly empty updates retain the last good snapshot; deletions need a visible preview/policy. An update must not silently reconnect the current session. Support ETag/Last-Modified when available. Scheduled refresh is opt-in and must have an explicit direct/proxy fetch path.

TLS verification stays enabled. Reject or explicitly mark insecure imported settings; never silently add insecure flags to fix connectivity.

## 9. Storage, secrets and update security

Use XDG locations on Linux, platform-appropriate Application Support locations on macOS, and per-user application data on Windows. Maintain separate config, state, cache and runtime directories. Use 0700 directories / 0600 secret files on Unix and corresponding Windows ACLs. Protect durable recovery journals from unprivileged tampering.

Start with schema-versioned atomic files and locks; do not add SQLite until concurrent/query requirements justify it. Use temporary files in the same filesystem, fsync and atomic replacement where supported. Migrations preserve backups and cannot silently lose profiles. Keychain/credential-manager storage is optional later; permission-restricted plaintext is not encryption at rest and must be described honestly.

Redact subscription URLs, passwords, UUID-like credentials, private keys, auth headers and raw configs from logs, errors, JSON and support bundles. Secret-safe logging is required at creation time, not only during export. Debug logging needs explicit scope and still excludes credentials. No telemetry by default. Speed tests, health checks and external IP checks disclose their destination and bandwidth use; no silent third-party connectivity calls on startup.

Core updates are explicit, staged and version-pinned. Verify artifact integrity against a trusted release manifest; checksums from the same untrusted download channel alone do not prove authenticity. Verify signatures/provenance where supplied. Defend against archive traversal, symlinks and executable substitution. Keep the last known good core. Never replace a running executable or mutate a privileged install through user-controlled paths. Bundle necessary Windows runtime/driver components only after compatibility and distribution review.

## 10. Platform support policy

Do not promise all Linux distributions. Publish tested combinations and minimum OS/kernel/runtime requirements derived from tests.

| Platform | Initial release scope | Later scope / conditions |
|---|---|---|
| Linux amd64, arm64 | Foreground CLI/proxy; validated TUN on selected distros | More distributions after DNS/firewall/init testing |
| Linux without systemd | Foreground operation | OpenRC/runit integration separately |
| macOS arm64, amd64 | CLI/proxy after native tests | TUN, resolver handling and launchd after dedicated validation |
| Windows amd64 | CLI/proxy in PowerShell and Windows Terminal after native tests | TUN, DNS, firewall, service and elevated lifecycle testing |
| Windows arm64 | Experimental until artifacts and dependencies are validated | Promote after native CI |
| WSL / containers | Proxy where tested | TUN unsupported unless explicitly certified |

Candidate Linux matrix: Ubuntu LTS, Debian stable, Fedora, Arch, openSUSE and Alpine. Pin exact versions and kernels in CI. Alpine/musl support must be tested for both the app and the selected core. Add representative NetworkManager/systemd-resolved and non-systemd scenarios. PowerShell is a command shell, not a substitute for Windows networking APIs.

Distribute signed/checksummed archives first; add deb/rpm/Homebrew/Windows packaging after lifecycle tests. No required `curl | sudo sh`. Installation must not connect, change DNS, open firewall ports or enable boot persistence. Uninstall stops owned sessions, removes only owned service/resources and preserves profiles unless purge is explicit.

## 11. Verification and performance gates

Unit/fuzz tests: share-link parsers, bounded decoding, canonicalization, redaction, terminal escape sanitization, capability matching, network plans, journals, migrations and conflicting restores.

Integration tests: use pinned real cores and controlled protocol servers with synthetic credentials. Validate generated configurations and actual DNS/HTTP/TCP/UDP behavior as appropriate. Distinguish authentication failure, endpoint DNS failure, timeout, unsupported transport and local permission errors.

Privileged network tests MUST run in disposable VMs or suitably isolated namespaces. Do not modify the developer's or CI runner's real default route. Containers alone do not validate Windows/macOS or all Linux host network-manager behavior.

Release-blocking cases:

- Ctrl+C, SIGTERM, core crash, UI crash, supervisor crash, watchdog crash and disk-full journal failures.
- Kill at every transaction boundary; restart and repeated recovery are safe and idempotent.
- SSH management remains reachable or rollback occurs without the lost connection being required.
- Port/resource collisions, multiple users, PID reuse, malicious IPC and writable executable replacement.
- Other VPN/firewall coexistence, DNS manager changes, DHCP renewals, Wi-Fi/Ethernet changes and sleep/wake.
- IPv4/IPv6 packet capture proves the documented tunnel/bypass/block policy during activation, reconnect and failure.
- No credential or terminal-control injection in logs, errors, JSON, TUI or diagnostic exports.
- Stop/uninstall leaves no owned networking resources; concurrently changed unrelated resources remain intact.
- No boot service or network configuration changes from proxy-only commands.

Initial performance targets, not measured claims: warm simple CLI commands under 150 ms p95 on a documented reference host; idle controller CPU below 1%; controller RSS below 60 MiB excluding core; bound logs and import memory. Record hardware, sample count and core separately. Compare throughput/latency against running the same core directly; investigate overhead above 5% under a controlled repeated benchmark. A shell wrapper or language choice does not guarantee faster traffic processing.

Required evidence: tests run, exact environment, packet captures/assertions where relevant, remaining limitations, and tested compatibility manifest. Cross-compilation success is not functional certification.

## 12. Delivery plan

| Phase | Deliverable | Exit gate |
|---|---|---|
| 0 | Architecture decisions, threat model, compatibility manifest, CLI contract | Chosen pinned versions and safe VM test harness |
| 1 | Go CLI, private store, import, sing-box proxy foreground, duration, stop/status | Real proxy request; no host mutation; parser/redaction tests |
| 2 | Subscriptions, child `exec`, optional TUI, detached sessions | Atomic update, terminal/JSON usability, process ownership tests |
| 3 | Linux TUN planner, privilege helper, journal and watchdog | Fault injection, DNS/IPv6 and SSH rollback tests |
| 4 | Optional services, Xray adapter, diagnostics and update workflow | Backend matrix, uninstall and upgrade rollback evidence |
| 5 | macOS/Windows TUN and packaging | Native platform network/recovery tests |
| 6 | Advanced policies | Independent acceptance tests per feature |

Advanced backlog: split routing by domain/CIDR, supported process/UID routing, opt-in failover, latency probes, bandwidth-capped speed tests, opt-in subscription schedules, kill switch, LAN proxy with authentication, rule-set updates, encrypted backup, and additional protocols. Explicitly display unavailable features. Do not ship fake controls, fabricated metrics or silent fallbacks.

Do not estimate a reliable cross-platform TUN implementation as a one-prompt script. The fastest defensible path is a small functioning proxy release followed by independently gated network changes.

## 13. Agent execution rules

1. Treat this specification as requirements, not proof of existing implementation.
2. Implement one phase at a time, with small reviewable changes. Begin with Phase 0 then Phase 1.
3. Read the repository instructions and inspect existing code before changing it.
4. Verify dependency APIs and core schemas against pinned versions; do not copy stale examples or use guessed flags.
5. Use real core validation and controlled integration tests; mocks do not prove connectivity or rollback.
6. Never test privileged networking on a real workstation or production server without an explicitly approved test environment.
7. Do not weaken TLS, disable IPv6 globally, flush firewalls, modify unrelated routes, or run downloaded shell scripts to make a test pass.
8. Do not run the TUI/parser as root just to simplify TUN integration.
9. Keep all user-facing text English. Keep secrets out of examples, screenshots and commits.
10. Report what works, what was tested, limitations and the next phase; never claim untested platform support or zero risk.

### Copy-ready starter prompt

> Read TERMINAL_VPN_BUILD_SPEC.md and the repository instructions. Build a Go terminal-only proxy/VPN client with Cobra, optional Bubble Tea/Lip Gloss, and sing-box as the first external core. Start with Phase 0 and Phase 1 only. Choose and pin compatible versions from official sources, record the capability manifest, and implement profile import, private storage, generated-config validation, loopback-only proxy foreground sessions, duration deadlines, status and safe stop. Keep CLI/application/core/storage boundaries clear so TUI and TUN can be added later. Do not mutate host routes, DNS, firewall or boot services in this phase. Add meaningful parser, redaction and real-core integration tests using synthetic fixtures. All UI/help/errors are English. Document commands and verification evidence. Do not claim TUN, crash recovery or platform support until the corresponding gates pass. Finish the phase with a concise change summary, tests and known limitations.

### Repository documentation to produce

README.md; docs/architecture.md; docs/security.md; docs/cli.md; docs/compatibility.md; docs/core-versions.md; docs/recovery.md; docs/testing.md; docs/releasing.md; and decision records for storage, privilege separation and route ownership. Recovery documentation must be available locally/offline.

## 14. Official references and research boundaries

References consulted 2026-10-14. These identify upstream facts; architecture, limits, defaults, budgets and phased scope above are recommendations for this product. They are not upstream promises. Re-check the exact release before implementation.

- [S1] v2rayN: https://github.com/2dust/v2rayN — a GUI client managing cores; not the original V2Ray core itself.
- [S2] sing-box source/releases: https://github.com/SagerNet/sing-box
- [S3] sing-box TUN: https://sing-box.sagernet.org/configuration/inbound/tun/
- [S4] Xray TUN: https://xtls.github.io/en/config/inbounds/tun.html ; source/releases: https://github.com/XTLS/Xray-core
- [S5] Bubble Tea: https://github.com/charmbracelet/bubbletea
- [S6] Lip Gloss: https://github.com/charmbracelet/lipgloss
- [S7] Cobra: https://github.com/spf13/cobra
- [S8] Go build targets: https://go.dev/doc/install/source#environment
- [S9] v2rayN subscription description: https://github.com/2dust/v2rayN/wiki/Description-of-subscription

No source here establishes that this proposed application has been security-audited, built, benchmarked, or tested. Those are implementation deliverables.
