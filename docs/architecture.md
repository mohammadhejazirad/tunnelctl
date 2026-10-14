# Architecture and Phase 0 decisions

The specification is the product baseline. The initial scope is a client for
existing servers. Real proxy behavior is the first acceptance gate; later TUN
work has independent safety gates.

`cmd/tunnelctl` only connects process I/O to `internal/cli`. Cobra handlers parse
arguments and render results. `internal/app` supplies reusable profile/session
use cases. Domain models distinguish private Profile credentials from public
redacted profiles and Session state. `internal/importer` normalizes untrusted
SIP002 links. `internal/store` owns private files, atomic replacement and locks.
`internal/session` owns session orchestration and deadlines. `internal/core`
defines the backend/process contracts; `internal/core/singbox` renders, checks,
starts, verifies readiness and stops the pinned executable. Linux identity,
locking, peer credentials and child configuration live in `internal/platform`.

No empty future packages were created. The Bubble Tea v2/Lip Gloss v2 TUI calls
the same app services and maintains a separate presentation model. Async commands,
one-second refresh, bounded events, compact views, keyboard navigation and color/
nonterminal rules keep networking outside the view loop. It never changes host
networking; leaving the UI does not stop a detached session.

The adapter approves one exact binary digest and schema version. It holds the
verified executable inode open and executes `/proc/self/fd/N` with independent
arguments, so pathname replacement does not substitute another executable.
Same-user mutation of the executable inode remains outside the threat boundary.
Every generated config is validated with that exact core before launching the
listener. The config's pathname is unlinked immediately; the child reads inherited
descriptor 3. No raw imported configs reach the core.

One foreground or explicit detached supervisor holds a kernel flock per runtime.
Detached startup reexecutes the protected application with readiness and bounded
private request pipes; listener credentials never enter process arguments. Child
execution pins the running application inode instead of reopening its replaceable
installation pathname. Same-UID in-place executable mutation remains outside scope.
Child
exec creates an isolated runtime, direct child and ephemeral listener, without
touching the unrelated main session or global environment.
Status/stop use a private Unix socket, and both client and server check SO_PEERCRED
UID. Only the supervisor signals its live Go child Process handle. Durable session
metadata is informational, never authority to kill a PID. Linux procfs socket inode
checks plus SOCKS negotiation establish that the ready listener belongs to the
owned core. The run goroutine remains on its OS thread while the core lives so
Linux Pdeathsig remains bound to the living supervisor thread.

State progresses through preflight, prepared, starting, listener_ready, stopping
and idle. Process liveness and listener readiness are separate fields. External
request tests do not fabricate per-session connection verification. Stale metadata
without a live lock is cleanup_required. A held lock without reachable IPC is a
conflict and is never converted into permission to signal a stored PID.

Ordinary filesystem atomicity and flock are assumed on a local Linux filesystem.
Network filesystems and filesystems lacking Unix modes are unsupported. Storage
schema 3 has backup-preserving schema-1/2 migrations; unknown newer schemas fail
without overwriting content. Subscription source/profile updates use one file,
strict bounded parsing, per-source locks and revision CAS. See subscriptions.md.
Preferences and user metadata use this same snapshot/lock. Multifield TUI drafts
commit with revision CAS through shared app services; see storage.md. Runtime
status, cache and future privileged journals are not stored as connection truth.
The `--data-dir` override replaces the proposed generic `--config` in Phase 1;
it selects all four private directory roots consistently rather than introducing
an unimplemented application-config file format.

See ADRs for storage, privileges and route ownership. Future adapters must provide
their own capability matrix and native ownership tests.
