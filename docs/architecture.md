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

No empty future packages were created. TUI will call app services and maintain a
separate presentation model. Network operations will produce asynchronous events;
bounded logs, small-terminal rendering, keyboard access, NO_COLOR and TERM=dumb
are Phase 2 requirements. Bubble Tea/Lip Gloss are deferred dependencies: select
and lock compatible versions when actually implementing TUI, avoiding unused code.

The adapter approves one exact binary digest and schema version. It holds the
verified executable inode open and executes `/proc/self/fd/N` with independent
arguments, so pathname replacement does not substitute another executable.
Same-user mutation of the executable inode remains outside the threat boundary.
Every generated config is validated with that exact core before launching the
listener. The config's pathname is unlinked immediately; the child reads inherited
descriptor 3. No raw imported configs reach the core.

One foreground supervisor holds a kernel flock per private runtime directory.
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
schema 1 has no migration yet: unknown schemas fail without overwriting content.
The `--data-dir` override replaces the proposed generic `--config` in Phase 1;
it selects all four private directory roots consistently rather than introducing
an unimplemented application-config file format.

See ADRs for storage, privileges and route ownership. Future adapters must provide
their own capability matrix and native ownership tests.
