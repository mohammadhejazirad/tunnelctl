# Versions and schemas

Semantic Versioning 2.0: `vMAJOR.MINOR.PATCH` with optional prerelease such as
`-rc.1`. Numeric leading zeros and tag build metadata are rejected. `version --json`
reports application, schemas, core, commit/date, toolchain and target independently.

Source builds: `0.2.0-dev`, unknown commit/date, development=true. Release tool
derives version from tag, injects ldflags metadata and requires clean exact tagged
HEAD. Dirty dry-run commits end `-dirty`, development=true. Application commands
check version/tag consistency; archive tests compare installed metadata to the tag.
Dry-run never creates a tag or publishes.

Pre-1.0 minors may change contracts with migration/release notes; patches should
retain compatibility. Prereleases are explicit opt-in; no updater exists. 1.0 gates:
stable CLI/JSON, verified supported native targets, independent security/recovery
review and resolution of release blockers. Untested features cannot be promoted
merely to achieve 1.0.

Storage schema 3 commits preferences/metadata/sources/profiles together. Schema 1/2
remain readable. First successful mutation saves the exact original
`profiles.schema1.backup.json` or `profiles.schema2.backup.json`, then commits schema 3.
Failed mutation does not migrate; existing different backup blocks migration and
is never overwritten. Secret backups remain 0600. Newer schemas are refused without writing. Output envelope
schema remains 1; core is sing-box 1.14.3.

The actual 0.1 baseline's strict decoder reports code 2 (invalid storage) for new
schema-2 fields before reaching its schema-version check. It refuses without
writing; a dedicated old/new binary test verifies byte-for-byte preservation.

Downgrade: stop, privately preserve schema-3 data, restore a verified compatible backup
into a **separate** old-version data directory. Old binaries reject schema 3.
Later changes cannot be translated to an older schema automatically. Upgrade
migration tests are real data tests, not a claim of published-binary upgrade or
power-loss certification.

See [durable data](storage.md) for exact transaction, form conflict, restore and
explicit deletion behavior. Source version remains 0.2.0-dev; no published tag is
rewritten and no new public release is implied by the storage schema increment.
