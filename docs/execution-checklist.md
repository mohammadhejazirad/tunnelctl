# Product and release execution checklist

Baseline inspected 2026-10-14: commit a783dc6, clean worktree, no AGENTS.md and no
existing workflows. Real Linux-amd64 Shadowsocks proxy, private schema-1 storage,
JSON CLI, duration and stop/status work. Test-only lifecycle fakes are clearly
separate from real-core tests. No production TODO/mock feature is advertised.
Subscription, detached sessions, exec, TUI, version injection and release tooling
were absent. Cross builds do not provide native Windows/macOS support.

- [x] Inspect repository, full build specification, architecture and existing evidence.
- [x] Prepare isolated verified GCC 15 and run baseline/final full race with a race CLI.
- [x] Implement transactional subscriptions, HMAC previews, safe fetching/bounded parsing.
- [x] Test HTTPS success/failure, preview, concurrent stage conflicts, identity and recovery.
- [x] Implement schema migration with backup and incompatible downgrade refusal.
- [x] Complete profile rename/remove, inode-pinned detached proxy, child exec, logs/completion.
- [x] Implement shared-service TUI; verify compact/resize models and native PTY smoke.
- [x] Implement SemVer/build identity, tag validation, changelog and release notes.
- [x] Add pinned CI/draft workflows; validate actionlint, vulnerability and hygiene checks locally.
- [x] Build/extract five archives; verify metadata/install/uninstall and real packaged proxy.
- [x] Update README, actual compatibility matrix, security and offline recovery.
- [ ] Verify native targets/distributions before enabling their runtime support.
- [ ] Implement TUN only after an isolated privileged environment can certify journal,
      ownership, SSH rollback, watchdog, DNS/IPv6 and recovery gates.
- [ ] Enable Xray only after a pinned artifact and real protocol/transport tests.
- [ ] Certify explicit service install/uninstall in an isolated supported init environment.
- [ ] Publish only after explicit authorization to push/tag/publish.

External scope: origin is https://github.com/mohammadhejazirad/tunnelctl.git.
No push, tag publication, GitHub release publication or external repository settings
change is authorized in this turn. Draft-release workflow and local release dry-run
are authorized. Linux-only tested runtime remains the release support boundary until
native validation exists. Broader build artifacts must be labeled compilation-only.

Environment constraints: source filesystem fuseblk does not protect secrets; use
private Linux /tmp/home storage. Docker daemon access is currently denied. A working
VM/namespace gate must be demonstrated before any privileged network test.

## Completed local evidence

Expanded race including baseline upgrade passed: integration 142.950 seconds. Go test/vet,
gofmt, module verification and whitespace checks passed. govulncheck v1.9.0 reports
no known vulnerabilities; actionlint v1.7.12 validates workflows. Neither proves
an independent security audit or a successful remote GitHub run.

Initial archive install/extraction tests passed; final candidate archive tests
verify all five target formats, checksums, license/docs, independent CGO=0/target
metadata and native Linux version/tag. Real tests using the extracted final binary
passed in 7.968 seconds, repeated against delivered dist in 8.229 seconds. Manual
installation/replacement/removal preserves populated profiles. Actual upgrade from
the 0.1.0 development executable and safe old-binary refusal also passed. No prior
public release exists for a published-binary upgrade claim.
Go -trimpath intentionally omits -ldflags from debug metadata (Go issue 52372);
cross-target static checks inspect embedded strings, while native version-command
equivalence is a separate runtime gate, not inferred from those strings.

Dry-run candidate v0.2.0-rc.1 is development=true, commit a783dc6...-dirty. Final
deliverables are generated under ignored dist/verified-final/; earlier dry-run assets
remain at dist/ rather than being overwritten. User/core secrets never enter them.
No source commit or public release/tag was made. Core is externally user-installed.

## Remaining product and external gates

- TUN: unimplemented. No provisioned host-like disposable VM certifies the helper,
  journal, independent watchdog, SSH external confirmation, simultaneous manager
  changes, packet-level DNS/IPv6 and idempotent recovery. Rootless namespace/TUN
  device availability alone is insufficient. Host networking was never changed.
- Xray: unimplemented, not a fake adapter. Needs independent schema/artifact,
  listener/capability integration and real protocol/transport tests; missing work,
  not a claim that the core cannot support the desired protocols.
- Services: unimplemented; systemd/other init install/uninstall needs an isolated
  persistent init environment and crash/recovery tests, not host installation.
- Other OS/architectures/distributions: cross builds only; native storage/ACL/IPC,
  core approval and actual network/lifecycle tests remain. Remote release jobs
  check version/help only and do not promote unsupported proxy combinations.
- Publication: maintainer review/license sign-off, explicit commit/tag/push authority,
  real remote CI and draft/provenance review before any public publication.

This completes the tested phase-2 Linux proxy candidate and local release
preparation, **not the entire TUN/cross-platform/backend/service product**.
