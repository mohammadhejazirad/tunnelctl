# Releasing tunnelctl

Repository: https://github.com/mohammadhejazirad/tunnelctl. Source is 0.2.0-dev;
v0.2.0-rc.1 is an existing **dry-run candidate, not a public release**. Do not
create a public tag just to exercise CI.

## Before tagging

1. Review CHANGELOG, support claims, licenses, storage migrations and security
   gates. Only Linux amd64 proxy is approved. Other installers are experimental;
   TUN/services/Xray are unsupported, approved sing-box is external.
2. Run Quality gates and Packaging validation on the proposed commit. Manual
   Packaging validation / Release runs are always dry: no tag, draft or public
   release is created. Native Windows/macOS/ARM gates must actually pass.
3. Select an unused version, commit reviewed source, ensure a clean tree, then
   tag/push. Packaging supports vM.m.p and alpha.N/beta.N/rc.N (N=1..199), no build
   metadata. See [version limits/mappings](packaging.md). Example commands below
   are a template, **not** a selected/created next release:

```sh
git status --short
git tag -a v0.3.0-rc.1 -m 'tunnelctl v0.3.0-rc.1'
git push origin v0.3.0-rc.1
```

Stable requires a new independently reviewed tag such as v0.3.0, never an edited
RC binary. Never rewrite a published tag or replace public assets.

## Workflow contract

`quality.yml`: push/PR/reusable. Formatting, vet, tests, hygiene, mod verify,
govulncheck, historical executable/schema upgrades, real pinned core, separate
CGO=1 race builds, and five-target CGO=0 compilation.

`packaging.yml`: push/PR/manual/reusable. GoReleaser OSS 2.18.3 with embedded nFPM
2.47.0 builds five archives and eight Linux packages. appimagetool 1.9.1 builds
two digest-pinned images. Native jobs build/test WiX 5.0.2 MSI and macOS pkg.
Containers test Debian/Ubuntu, Fedora, Arch and Alpine without host installation
or networking changes. Native MSI tests cover collisions, PATH, repair, a
synthetic lower-version installer upgrade, downgrade refusal and removal. Native
pkg tests cover collision, install/reinstall, PATH and receipt-scoped removal.
No previous public installer is assumed. Arch ARM64 has payload smoke only:
there is no approved Arch Linux ARM installation environment in this matrix.

`release.yml`: pushed v* tags validate SemVer, clean exact-tagged commit, then
all quality and packaging jobs on that commit. Aggregate demands exactly 23
named payload/metadata assets and adds checksums.txt (24 uploads total). The
tag-only write job rechecks manifest/tag/commit and hashes, requests GitHub build
provenance, creates a draft, uploads the complete set, downloads/verifies it,
then **automatically publishes**. A failed gate cannot publish a partial matrix.
alpha/beta/rc have prerelease=true/latest=false; stable latest=true. No updater
silently moves stable users to prerelease. Tag versions chronologically; do not
push an old stable after a newer stable and change Latest.

Only the final job has contents:write/id-token:write/attestations:write. PR code
has no publication secret. Actions are commit-pinned; tools version/digest-pinned;
release concurrency prevents overlapping runs for one ref. Manual dispatch cannot
reach publication. No APT/DNF/AUR/Homebrew/WinGet repository is created.

Public staging only uses named documents/examples and linked-module license
texts, never a recursive checkout harvest. Each staged file has an explicit
archive/package destination and mode; no recursive checkout glob is packaged.
Private samples and harness reports are never read/uploaded. Docker context is
deny-all; installation containers receive only public assets read-only. CycloneDX
is a linked-Go-module inventory, not an external-core/OS filesystem SBOM.

## Local dry-run

Run the following orchestration on Linux amd64; native MSI/pkg jobs are separate.

```sh
bash packaging/tools.sh /tmp/tunnelctl-package-tools
export PATH="/tmp/tunnelctl-package-tools:$PATH"
go run ./cmd/packaging --tag v0.2.0-rc.1 --dry-run --out dist/packages
bash packaging/appimage/build.sh dist/packages /tmp/tunnelctl-package-tools 0.2.0-rc.1
bash packaging/test-archives.sh dist/packages/assets v0.2.0-rc.1
```

Use a new output directory. Dirty dry-runs carry -dirty commit/development=true.
Date derives from the source commit. GoReleaser's snapshot **packaging engine**
supplies mapped native versions; independent exact-tag validation, explicit
ldflags and disabled publisher control official builds. MSI/pkg need native
runners; finalize refuses missing files. Never add fake installers to pass it.
No byte-for-byte installer/AppImage reproducibility is claimed. The original
archive-only cmd/release remains a diagnostic regression path.

## Failure and retry

Failed upload/verification leaves at most a draft, never a public partial release.
Reruns refuse existing drafts **and** public releases; no --clobber is used.
Inspect any failed draft/run, explicitly delete only that unpublished failed
draft after maintainer review, then rerun the unchanged tag/commit. No automatic
draft/tag deletion occurs. Changed source requires a new commit/version/tag.
Public releases are immutable: fix with a new release, not replacement assets.
The public Go suite tests the real publisher script against a synthetic `gh`
transport, including both release channels and refusal/failure paths. This is
not a substitute for the first authenticated remote dry-run/native gates.

## External setup, signing and authenticity

No custom publishing secret is needed: the scoped GITHUB_TOKEN publishes to the
same repository. Actions policy must allow that job to write releases; tag
rulesets must permit the authorized tag push. Native runner availability and
attestation eligibility depend on the actual GitHub account. Attestation failure
is a release gate. No remote run is claimed by a local configuration check.

Current installers/packages are **unsigned** and macOS **not notarized**. SHA-256,
native package signatures, Windows Authenticode, macOS codesign/productsign,
notarization and GitHub provenance are different assurances. Verify provenance
when actually available:

```sh
gh attestation verify tunnelctl_VERSION_linux_amd64.tar.gz --repo mohammadhejazirad/tunnelctl
```

Future optional signing requirements (not wired or claimed functional):

- Windows: WINDOWS_SIGNING_CERT_PFX, WINDOWS_SIGNING_CERT_PASSWORD, certificate
  identity/approved timestamp URL; sign exe before MSI, then MSI with signtool;
  native verification must precede final hashes/provenance.
- macOS: MACOS_APPLICATION_CERT_P12, MACOS_INSTALLER_CERT_P12,
  MACOS_CERT_PASSWORD, APPLE_ID, APPLE_TEAM_ID, APPLE_APP_SPECIFIC_PASSWORD (or
  reviewed notary API key); ephemeral keychain, hardened-runtime codesign,
  productsign, notarytool submit/wait and staple/verify, before final checksums.
- Linux: reviewed GPG/APK package-key management and public trust instructions.

Adding secrets alone does not enable these stages. Implement/test/gate signing
before advertising it. Never recommend disabling Gatekeeper, SmartScreen or
system security enforcement; use reviewed source builds where policy blocks
unsigned artifacts. No signing credentials or public release were fabricated.
