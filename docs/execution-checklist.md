# Product and release execution checklist

## Private testing and menu increment — 2026-10-14

- [x] Read the complete build specification and inspect existing shared services/CI.
- [x] Check exact .gitignore, private input ignore rule, index and reachable history.
- [x] Protect release inputs with explicit public document/example lists; deny Docker context.
- [x] Run the baseline race suite with the existing isolated C toolchain.
- [x] Implement bounded, opt-in anonymous private test harness; never read samples on startup.
- [x] Implement menu/forms, cancellation, profile/subscription flows and connection settings.
- [x] Test noninteractive startup, model flows, PTY, real core and complete race.
- [x] Execute bounded authorized private tests; no private-derived report is stored in the repo.
- [x] Rebuild and inspect release archives; synchronize README and test evidence.
- [ ] TUN isolated acceptance: unavailable; never mutate the real host network.

Public controlled menu-increment evidence: go test/vet/audit/actionlint pass.
The first full race run with race CLI + development harness/pinned core passed
(integration 155.607 seconds), and ordinary real-core acceptance passed in
12.963 seconds. Five initial menu archives passed membership/checksum/metadata
and Linux installation checks. An overbroad test initially rejected the *public*
private-testing guide by substring; the assertion now distinguishes documentation
from private inputs/reports/executables. No actual private file was packaged.
Final review added empty-session confirmation protection and CLI equivalents of
menu config-check/probe diagnostics. Subsequent final gates are recorded below.

Second complete race passed (integration 155.478 seconds). Final CLI config-check/
TLS-probe diagnostic acceptance passed with a rebuilt race binary (12.839 seconds);
all shared internal packages passed race again. Latest ordinary full real-core
suite passed (integration 13.885 seconds). govulncheck reports no known vulnerabilities;
actionlint validates both pinned workflows. Historical upgrade acceptance was skipped
in these runs because TUNNELCTL_PREVIOUS_BIN was not set; earlier baseline evidence
is retained below, not represented as a new upgrade run.

Delivery for this increment: ignored dist/menu-delivery/, fresh v0.2.0-rc.1 dry-run
assets, commit 2d749d6...-dirty, development=true. All five archives passed checksums,
public-only membership and target metadata; native Linux installation/version gate
passed in 0.326 seconds. Extracted final Linux binary passed controlled real-core,
TLS/probe, detached/foreground, PTY, duration/signals/crash/cleanup and synthetic
private-harness acceptance in 14.212 seconds. Final ordinary full suite passed;
latest full public race (optional real-core variables unset) passed separately.
The final shared diagnostic addition passed targeted real-core race, and the
last harness/refetch/render assertion review passed related package race.
Private reports and the development harness executable are absent from assets.
No public release/tag/push or native non-Linux runtime test was performed.
The preceding archive evidence belongs to dist/menu-verified/ before a final
bounded error/remedy wrapping improvement. dist/menu-delivery/ is rebuilt from
the final implementation; no prior dry-run asset is overwritten. Compact-size
tests cover message bounds down to 16 columns / 5 rows; standard-sized views
reserve space for both the error and its remediation instead of hiding the latter.
Final delivery gates completed: all five dist/menu-delivery/ archives passed
membership/checksum/native metadata checks (0.334 seconds); extracted Linux binary
passed the complete controlled network/PTY/lifecycle/harness suite (13.985 seconds).
The last UI wrapping/compact-size review passed related model/CLI race, vet,
formatting, repository hygiene and whitespace checks. Final published status: none.

Private outcomes are intentionally absent here and in package documents. Use the
explicit development harness; do not upload even anonymized private-derived
reports. Private input bytes were never modified or made into public fixtures.

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

Earlier baseline dry-run candidate v0.2.0-rc.1 was development=true, commit a783dc6...-dirty.
Its deliverables were generated under ignored dist/verified-final/; earlier dry-run assets
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
