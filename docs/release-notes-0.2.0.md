Development candidate for Linux amd64 terminal proxy clients. The package uses
the user's separately installed, verified sing-box 1.14.3; no core is bundled.

Includes transactional subscriptions, protected previews/last-good snapshots,
schema migration backup, profile management, detached sessions, isolated child
exec, keyboard TUI, private state events and completion. See CHANGELOG.md and
docs/testing.md for actual evidence and limits.

Only Linux amd64 proxy runtime is tested by this local acceptance environment.
Other archives are explicitly labeled compile-only. TUN, system services and Xray
are unsupported, with no unsafe placeholder controls. No public release has been
published by the local dry-run. Native CI results and provenance are not fabricated.
