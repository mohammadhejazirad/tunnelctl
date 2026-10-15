# Durable user data and recovery

## Contract and contents

The supported Linux runtime uses one authoritative **schema-4** snapshot,
`config/profiles.json`, with an independent monotonic revision. CLI and TUI call
the same application services and writer transaction. Successful import, select,
rename, metadata, source update/removal or settings Save means the snapshot was
written, file-synced, atomically renamed and parent-directory-synced. Failure is
reported, never silently replaced with empty settings. A directory-sync failure
after rename reports code 7: replacement may already be visible but durability
is uncertain; inspect/reload rather than assuming rollback.

The snapshot persists normalized complete supported Shadowsocks connection
parameters, passwords, stable IDs, custom names, tags, favorites, selected profile,
subscription ownership, protected source URLs, CIDR/CA trust configuration,
ETag/Last-Modified, source revision and preview key. Raw share links are not stored.
Subscription refresh preserves user metadata and identity. Removing a source
retains its profiles as standalone; profile removal is separate.

Preferences persist listener address/port, default duration, authentication-file
reference, preferred mode, core backend/path, readiness timeout, color preference,
explicit probe target and manual subscription policy. Authentication-file contents
are not duplicated in preferences: preserve that private file as well. Empty core
path means `TUNNELCTL_CORE`, then `/usr/local/bin/sing-box`. Paths saved as defaults
must be absolute so a different working directory cannot change their meaning.

Only `127.0.0.1`, Proxy, sing-box and manual refresh are supported. TUN, custom
host routing/DNS/IPv6 policies, automatic connection, scheduled refresh, alternate
backends and service boot settings are unavailable and rejected, not dummy saved
controls. Proxy leaves host routing/DNS/IPv6 unchanged; application proxy support
and endpoint bootstrap resolution still determine actual traffic/DNS behavior.

## Editing, precedence and concurrent processes

```sh
tunnelctl paths --json
tunnelctl settings show --json
tunnelctl settings set listener-port 2081
tunnelctl settings set duration 10m
tunnelctl settings set core-path /absolute/verified/sing-box
tunnelctl settings set auth-file /absolute/private/proxy-auth.json
tunnelctl settings set color false
tunnelctl profile metadata PROFILE_ID --tags work,preferred --favorite
tunnelctl settings reset --yes
```

`settings set --help` lists supported keys. Metadata replaces **both** tag and
favorite fields; omitted tags/favorite clear them. Tags are unique printable
1–64-byte strings without commas, up to 32 per profile. Public profile output
includes tags/favorite, never connection credentials.

TUI Settings/Connect opens one draft containing port, duration, auth-file reference,
hidden probe destination, core path, timeout and color. Arrows/Tab select fields,
Enter advances to **Save**, Enter on Save commits all fields together. Esc before
Save discards everything. Opening a form captures its revision. If another command
or menu changes the snapshot, Save fails with code 6 and asks for reload; it never
silently overwrites that process. Save already submitted is an atomic operation:
cancelling background work cannot undo a commit that completed. Success appears
only after commit. A pre-replacement failure retains prior defaults. If replacement
completed but directory synchronization failed (code 7), reloading can show the new
snapshot while reporting uncertain durability; it must not be mistaken for rollback.

Single-field CLI edits read the latest snapshot under the writer lock. Competing
writers either commit serially or receive explicit lock-conflict code 6; retry
after the other operation finishes. Do not delete a lock file to bypass a lock.
Settings reset requires `--yes` and preserves profiles, sources and selection.
`init` preserves existing data, including settings; it may complete a migration.

For `run` and `start`, saved defaults apply unless the corresponding flag is
explicit (`--listen-port`, `--duration`, `--proxy-auth-file`, `--mode`). Even an
explicit `--duration 0s` or `--proxy-auth-file ''` overrides a saved value for that
invocation only. `--core` and `--timeout` override stored defaults for CLI/TUI;
`NO_COLOR`, `--plain` and `--no-color` override the saved color preference.
Command flags never implicitly change preferences. `exec` intentionally uses its
own ephemeral listener/session, not the saved listener port/deadline.

Saved settings are **defaults for future sessions**, not applied-session state.
Save/import/select/update/init/startup do not disconnect, switch or auto-connect.
An existing supervisor keeps its captured core, listener, credentials and deadline.
Reconnect explicitly to apply those changes. Search filters, draft text, probe
results and navigation are temporary UI state, not preferences.

## Paths and security

Run `tunnelctl paths` as the original user (works even if profiles.json is corrupt).
`--data-dir /absolute/private/root` deliberately overrides the four roots.

| Linux data class | Default path |
|---|---|
| Persistent configuration/credentials/backups | `${XDG_CONFIG_HOME:-$HOME/.config}/tunnelctl` |
| Bounded session event history | `${XDG_STATE_HOME:-$HOME/.local/state}/tunnelctl` |
| Rebuildable cache (currently no essential user data) | `${XDG_CACHE_HOME:-$HOME/.cache}/tunnelctl` |
| Session IPC/metadata/locks | `$XDG_RUNTIME_DIR/tunnelctl`, otherwise state directory + `/runtime` |

Relative XDG values are ignored. Default paths never depend on CWD, executable
location, install prefix or archive extraction directory. Different users use
their own home/XDG roots; files additionally require matching UID ownership.
Do not intentionally point several users at a shared override. Unix directories
are 0700; authoritative, backup and staging files are 0600. Symlinks and permissive
or foreign-owned files are refused. Local Linux filesystems with flock, fsync and
atomic rename are required; network/FAT/fuseblk storage is unsupported.

**Permission-protected plaintext is not encryption at rest.** No fixed key or
Keychain dependency exists; headless servers use the same file protection. Protect
disk/account/backups separately. A compromised same-UID process can read these
secrets. No secret data belongs in Git, support logs, CI, or release archives.

macOS and Windows remain compile-only: Application Support/Cache and per-user
AppData/ACL implementations are not enabled or natively certified. They refuse
runtime storage rather than creating insecure fallback files. Linux arm64 builds
are also not runtime-certified. Do not interpret this document as cross-platform
persistence certification.

User storage refuses changed real/effective UID and sudo/pkexec environment
markers. Run CLI/TUI as the original user; do not create/chown user configuration
through elevation. No privileged helper or network recovery journal exists in
this Proxy release. A future helper needs separately owned machine-scoped journal,
not write access to the user's credentials. Persisted session metadata is only a
hint; status reconciles locks, authenticated IPC and process/listener identity.
Reboot never treats saved `connected` as fact and never reconnects automatically.

## Backup, migration and corruption

Stop sessions, close other writers and copy `profiles.json` to a private backup
location. Preserve any separately referenced proxy auth/CA files. These commands
use **placeholders**: substitute the exact paths from `paths`, not a broad XDG root.

```sh
tunnelctl paths --json
tunnelctl stop
install -d -m 0700 /absolute/private/tunnelctl-backup
install -m 0600 -- /absolute/config/tunnelctl/profiles.json /absolute/private/tunnelctl-backup/profiles.json
```

All backups contain secrets, including source tokens. Do not upload them.
The snapshot's connection model, app version, output schema and core version are
independent. Schema 1/2/3 can be read without writing; missing historical preferences
get conservative defaults. The first successful mutation makes an exact original
`profiles.schema1.backup.json`, `profiles.schema2.backup.json` or
`profiles.schema3.backup.json`, synced with private permissions **before** committing
schema 4. An existing equal backup is reused;
a different/insecure backup blocks migration without overwriting either file.
Failed validation or a canceled draft does not migrate data.

Unknown/newer schema, invalid settings, duplicate JSON fields, invalid references,
oversized/corrupt JSON and unsafe permissions are errors, never empty-store
fallbacks. Older strict decoders reject schema 4 (possibly code 2 rather than 3)
and must not rewrite it. Downgrade requires a compatible verified backup in a
**separate private data directory**; newer changes cannot be translated back
automatically. Keep the original damaged/newer file privately before any repair.
Validate a restored copy in a separate data root with `profile list` and
`settings show` before replacing the real file with all processes stopped.

Atomic writes use a same-directory 0600 `.atomic-*` stage, file fsync, rename,
then directory fsync. A killed writer leaves either complete old or new snapshot;
abandoned stages are private and never loaded. Inspect/remove their exact names
only with all writers stopped. Cleanup errors are surfaced. Physical reboot,
power loss and faulty disk/controller durability are not certified by SIGKILL
tests; durability relies on the filesystem honoring fsync/rename guarantees.

## Upgrade, uninstall and explicit complete removal

Archive update replaces only the executable; per-user data stays outside the
installation. Back up first. Ordinary manual uninstall stops owned sessions and
removes the exact installed executable; **it does not remove user data**. There
are no deb/rpm/AppImage/service installers with additional purge behavior.

No automatic purge command is provided. For deliberate complete removal:

1. Obtain exact directories with `paths`, stop sessions, close every CLI/TUI writer.
2. Decide explicitly whether to retain a private backup (a backup retains secrets).
3. Inspect files/types/ownership in only those application directories. Delete
   `profiles.json`, each schema backup and abandoned `.atomic-*` file by its exact
   inspected name, using an interactive confirmation such as
   `rm -i -- /absolute/config/tunnelctl/profiles.json`. Never use recursive removal
   against home, XDG roots, project or an unresolved variable/wildcard.
4. Similarly inspect/remove owned state/cache/runtime files after processes stop;
   remove empty application directories only. Separately decide whether to remove
   referenced auth/CA files and backup copies; they are not implicitly owned here.

This is distinct from `settings reset --yes`, which leaves credentials intact.
Deletion is not cryptographic secure erasure on SSDs/snapshots/backups.

## Reproducible public acceptance

`go test ./...` uses synthetic fixtures only. Storage tests spawn separate writer
processes, kill at create/fsync/rename/directory-sync boundaries, inject a real
RLIMIT_FSIZE write failure, check permissions and verify 50 concurrent commits.
They cover schema-1/2 backups, corrupt/newer refusal, XDG/CWD isolation, elevation
refusal, metadata and settings reset. CLI/model tests cover shared Save/reload,
discard, failed Save, stale-form conflicts and no automatic connection.
Real-core integration additionally checks PTY Save -> CLI read, process restart
using saved defaults, real proxy traffic and active-session isolation. Set
`TUNNELCTL_SCHEMA2_BIN` to a previous schema-2 executable for actual executable
upgrade/old-decoder refusal (optional; otherwise explicitly skipped).
No real private sample data or privileged network changes are involved.
