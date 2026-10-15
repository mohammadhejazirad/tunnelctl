# Changelog

## 1.1.0 - 2026-10-15

- Add clipboard-first import to the terminal UI, including automatic clipboard
  detection and a hidden manual-paste fallback when clipboard access is unavailable.
- Import individual `ss://`, `vless://`, `vmess://` and `trojan://` links, plus
  multiline/Base64 feeds and HTTPS subscription URLs.
- Support TCP, WebSocket, gRPC, HTTP and HTTP Upgrade transports, including TLS
  and Reality configuration with unsafe TLS verification bypass rejected.
- Upgrade persistent storage to schema 4 while preserving an exact private
  schema-3 backup during migration.
- Validate Shadowsocks, VLESS, VMess and Trojan with the pinned sing-box 1.14.3
  core, and ship installable packages for Linux, macOS and Windows.

## Unreleased packaging increment

- GoReleaser 2.18.3 / embedded nFPM 2.47.0: five CGO-free archives and Linux
  amd64/arm64 deb/rpm/Arch/apk; exact-hash PKGBUILD and pinned-runtime AppImages.
- Native WiX 5.0.2 MSI and macOS pkg build/install lifecycle gates, owned PATH
  entries/collision handling, no core/network/service side effects or data purge.
- Monotonic SemVer-to-native version mapping; complete expected-asset/hash gates,
  module SBOM, provenance and draft verification before automatic tag publication.
- Explicit experimental non-approved runtimes, unsigned/not-notarized installers,
  apk collision backup behavior and AppImage stable no-FUSE extraction guidance.

## Unreleased durable-storage increment

- Schema 3: persistent CLI/TUI defaults, profile tags/favorites and exact protected
  schema-1/2 backups; newer/corrupt data never resets silently.
- Atomic multifield Save/discard with stale-form detection, settings/paths commands,
  explicit per-run overrides and no implicit active-session changes.
- Separate-process concurrent writers, SIGKILL atomic-boundary recovery, actual
  failed writes, restart/CWD and executable schema-2 upgrade acceptance tests.
- Documented backup, recovery, settings reset and explicit complete data removal.

## Unreleased menu/private-testing increment

- Automatic TTY main menu, hidden/file imports, profile details/search/CRUD and
  subscription hidden add/preview/apply/rename/remove through shared services.
- Menu-local duration/port/authentication/probe settings, explicit detached exit
  options, cancellable background operations and confirmed-session-bound stop.
- Bounded opt-in anonymous development harness; private inputs/reports excluded
  from Git candidates, explicit release lists and denied Docker build context.
- Typed sanitized network failure categories and correct schema-2 diagnostics.
- Synthetic TLS transaction/menu/cancellation, PTY and real-core harness tests.

## 0.2.0 — Unreleased

- Transactional HTTPS subscriptions with previews, conditional requests, strict
  size/count limits, private CIDR exceptions and DNS-rebinding defenses.
- Schema 2 migration with a schema-1 backup and incompatible downgrade refusal.
- Profile rename/remove, explicit detached proxy sessions and isolated child exec.
- Bounded private state events, shell completion and a keyboard terminal interface.
- Semantic build metadata and a local archive/SBOM/checksum release builder.
- Linux race, real-core tests and pinned draft-release workflows.

TUN, Xray, services and native non-Linux runtime are not certified in this version.

## 0.1.0 — Development baseline

- Linux amd64 foreground loopback proxy using pinned sing-box 1.14.3.
- Private profile import/storage, redacted JSON, duration, status and safe stop.
- Real controlled Shadowsocks, HTTP/SOCKS and TLS CONNECT acceptance tests.
