# Release preparation and approval

The local Go release builder is the chosen packaging tool instead of GoReleaser:
small auditable archive policy, exact-tag gate, no publishing capability and no
additional runtime/build service dependency. It actually builds five target archives,
Unix tar.gz / Windows ZIP, with notices/docs/examples, module license texts,
CycloneDX module-inventory SBOM, checksums, notes and release manifest.

Only Linux amd64 is runtime-tested. Other targets have `_compile-only` names and
SUPPORT.txt warnings; build success never certifies a client platform. Core is not
bundled. Source license is Apache-2.0; maintainer must confirm authority/license
choice and third-party notices before public distribution. No external core license
is altered; its GPL distribution/notice obligations belong to separate installation.

```sh
go run ./cmd/release --tag v0.2.0-rc.1 --dry-run --out dist
TUNNELCTL_RELEASE_DIR="$PWD/dist" TUNNELCTL_RELEASE_TAG=v0.2.0-rc.1 go test -v ./test/release
```

Output must be new/empty and existing assets are never overwritten. SOURCE_DATE_EPOCH
can pin build date; byte-identical archive reproducibility is not yet promised.
RELEASE_GO can select the verified Go executable. Release builds use CGO=0; race
tests require CGO=1 and supported compiler separately. Dirty dry-runs are visibly
marked; no tag/push/publication occurs. Never distribute a dirty dry-run as stable.

Normal release builder requires clean HEAD at exactly the validated supplied SemVer
tag. Tag/binary metadata disagreement is rejected. Keep app/storage/core versions
independent. Maintain CHANGELOG.md and select explicit release notes; the current
builder targets docs/release-notes-0.2.0.md. See versioning.md for prereleases.

quality.yml checks fmt, vet, modules, tests, repository hygiene, real pinned core,
race-instrumented CLI, vulnerability scan, portable race and cross builds. Actions
are commit-pinned. The hygiene scanner is intentionally narrow, not exhaustive
secret detection; review changes before release. actionlint is pinned for workflow
validation. Remote CI has not been executed by this local development session.

release.yml triggers on v* tags, validates actual SemVer/exact clean HEAD in the
builder, depends on every quality job, builds all artifacts in one gated job,
checks archives/installed metadata and real extracted-binary proxy lifecycle, then
uploads assets. Only the final job gets contents:write/id-token/attestations. It
also waits for native version/help metadata checks on ARM64 Linux, both macOS
architectures and Windows/PowerShell. Those checks do not certify proxy runtime
or promote compile-only packages; local native runs remain unavailable.
checks transferred checksums, requests provenance and creates a **draft** with the
prerelease marker as appropriate. It refuses an existing release and does not
overwrite assets. A failed gate cannot publish a partial supported matrix.
Provenance availability is dependent on real GitHub execution/account policy;
no offline dry-run attestation is claimed.

Final maintainer actions, requiring separate explicit authority:

1. Review changes, license, release notes and unresolved support/security gates.
2. Commit the reviewed source; run remote PR quality and native portable runners.
3. Authorize/create/push a new valid release tag; never rewrite an existing tag.
4. Inspect draft checksums/SBOM/provenance and installed-user journeys.
5. Manually publish only the approved draft. Stable is not auto-promoted from RC.

No push/tag/public release or external repository-setting change was performed.
Do not advertise TUN, DNS-leak protection, kill switch, universal OS support or
unmeasured performance. Future supported targets must pass their own native gates.
