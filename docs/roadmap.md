# Roadmap

Phase 1 is Linux-amd64 foreground CLI proxy with narrow Shadowsocks import.
Phase 2 now implements transactional subscriptions with protected last-good snapshots,
controlled HTTPS fetching/SSRF defenses, child exec, a shared-service TUI and
explicit detached supervisors. Bubble Tea v2.1.0/Lip Gloss v2.0.6 are pinned after
official API checks. Local Linux tests include race, real HTTPS/core and PTY.
The 0.2 release preparation path includes archives, metadata and gated drafts.

Phase 3 adds a separately validated Linux TUN planner, narrow privilege helper,
durable resource journal, watchdog, management-route protection, external SSH
confirmation and timed rollback. DNS/IPv6 policy and failure behavior require
isolated fault-injection and packet-level tests. Host networking is not in scope
for Phase 1 or its tests.

Phase 4 adds independently tested Xray adapters, services, diagnostics and explicit
signed/versioned core update workflows. Later phases cover native macOS/Windows
storage/IPC/process ownership, TUN and packaging. Cross compilation does not
authorize or certify native runtime support. Secret export, richer error
classification and optional explicit connection probes are also deferred.

Next product gate: establish disposable Linux VM and adversarial network harness
for the privileged helper/journal/watchdog/SSH/DNS/IPv6 recovery boundary. Rootless
namespace availability alone does not certify real host managers, remote management
or service persistence. No TUN placeholder is exposed pending those gates. Xray
also needs independent approved artifacts, documented schemas and real transport
tests before enabling a second backend; no unused or fake adapter is added.
