# ADR 1: Private atomic JSON storage

Accepted for Phase 1. Use schema-versioned JSON with 0700 directories, 0600 files,
O_NOFOLLOW, UID checks, a profile-write flock, fsync of the replacement file, rename
in the same directory, and directory fsync. Readers observe a complete old/new file.
No SQLite or encryption claim is needed at this scale. A failed mutation retains
the prior state. Unsupported schemas require a compatible application or an explicit
schema migration with backups. Schema 2 uses one authoritative file for profiles
and subscriptions; see versioning.md for schema-1 backups and downgrade refusal.
Interrupted atomic writes may leave private staging
files; they are never loaded as the authoritative profile store.
