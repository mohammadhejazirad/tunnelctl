# Roadmap

Phase 1 is Linux-amd64 foreground CLI proxy with narrow Shadowsocks import.
Phase 2 adds transactional subscriptions with protected last-good snapshots,
controlled HTTPS fetching/SSRF defenses, child exec, a shared-service TUI and
explicit detached supervisors. Bubble Tea and Lip Gloss versions will be selected
from official release/API documentation at implementation time. No dependency or
fake command currently advertises those capabilities.

Phase 3 adds a separately validated Linux TUN planner, narrow privilege helper,
durable resource journal, watchdog, management-route protection, external SSH
confirmation and timed rollback. DNS/IPv6 policy and failure behavior require
isolated fault-injection and packet-level tests. Host networking is not in scope
for Phase 1 or its tests.

Phase 4 adds independently tested Xray adapters, services, diagnostics and explicit
signed/versioned core update workflows. Later phases cover native macOS/Windows
storage/IPC/process ownership, TUN and packaging. Cross compilation does not
authorize or certify native runtime support. Profile removal/export, richer error
classification and optional explicit connection probes are also deferred.
