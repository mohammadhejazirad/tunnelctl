# Packaging delivery checklist

This is evidence, not a claim that remote CI or untested operating systems passed.

- [x] Inspect name, clean baseline, version contract, CGO and external core policy.
- [x] Verify private sample is ignored and absent from the index; do not read it.
- [x] Select GoReleaser 2.18.3 (embedded nFPM 2.47.0), appimagetool 1.9.1,
  WiX 5.0.2 and native macOS pkgbuild/productbuild.
- [x] Implement/test native version conversion, clean-tag binding and public staging.
- [x] Build five portable targets and eight Linux packages plus two PKGBUILDs.
- [x] Build two AppImages with digest-pinned runtimes, notices/source/relink materials.
- [x] Implement native MSI/pkg collision, PATH and lifecycle gates (local execution unavailable).
- [x] Gate publication on all 23 assets + checksums and required native jobs.
- [x] Publisher synthetic-transport tests: stable/prerelease flags, existing
  release/API/upload failures, tampered download and manual-run refusal.
- [x] Validate workflows; public tests, vet, full race and real-core extracted payload passed.
- [x] Offline amd64 native lifecycle: Ubuntu 24.04.5, Debian 13.7, Fedora 43,
  Arch 20261011.0.609700 and Alpine 3.22.6. makepkg built the generated PKGBUILD
  as an ordinary user; its resulting package passed install/help/version/removal.
- [x] Explicit per-file archive/package destinations and modes; no flattened
  globs or private inputs. Native content gates check directory/document modes.
- [x] Root-protected macOS payload and trusted script PATH; native execution pending.
- [x] Document install/update/removal, signing and remaining evidence.
- [ ] Run remote GitHub dry-run and native Windows/macOS/ARM installation gates.
- [ ] FUSE-mounted AppImage runtime: /dev/fuse absent here; stable extraction passed.
- [ ] Prior public installer upgrade: none available (synthetic MSI fixture in CI).
- [ ] Signing/notarization: no certificates, stages intentionally not claimed enabled.

Observed/fixed: GoReleaser dist is not templated; archive root-file glob omission;
recursive glob flattening and directory mode inheritance; minimal Docker images
exclude documentation (test-only tunnelctl-scoped inclusion checks full contents);
secondary package-manager ownership queries must not erase the true owner;
Fedora merged sbin alias; apk non-aborting script collisions (private backup/warning);
AppImage transient parent signal semantics (sessions explicitly refused, stable
extraction real proxy/signals/crash/detached tests passed). Finalize on incomplete
local set correctly refuses missing macOS pkg, not a successful full dry-run.
Private file is ignored, untracked and absent from inspected history; never read.

No public version is selected here. The existing v0.2.0-rc.1 candidate is used only
for local/manual dry-runs. Linux amd64 remains the only approved proxy runtime.
