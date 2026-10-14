# Offline recovery

Normal stop: press Ctrl+C in the foreground terminal, or run `tunnelctl stop`
from another terminal as the same user with the same --data-dir (if overridden).
Repeated stop is safe. SIGTERM/SIGHUP behave like Ctrl+C. Closing a terminal only
delivers this behavior if the terminal/shell actually sends SIGHUP. Detached
execution, nohup and service lifecycles are not supported contracts.

If the core crashes, run fails with code 5 and cleans its socket/metadata. If the
supervisor dies abruptly, Linux Pdeathsig stops the core; stale nonsecret metadata
may remain. Status then reports cleanup_required. Run `stop` to remove the exact
owned control.sock and session.json; the persistent session.lock file is harmless
and should not be removed while an operation might own its inode.

A held lock with unavailable IPC reports a conflict. Retry briefly or stop the
foreground terminal. The application never infers ownership from a persisted PID,
never kills by executable name, and never signals an unrelated process to recover.
If code 7 reports an unexpected file at an owned resource name, inspect its type,
owner and origin manually. The program leaves ambiguous files intact. Resolve the
specific conflicting resource before retrying; do not delete an entire XDG root.

Secret config files normally have no pathname while running. Interrupted profile
updates may leave 0600 `.atomic-*` staging files inside the private config directory.
They are ignored. Keep a private backup of profiles.json before manual repair;
restore only a known good schema-1 file. A staging file is not automatically a
valid backup. No automatic migration or repair overwrites malformed storage.

There are no routes, DNS settings, firewall rules, services or shell configuration
to roll back in Phase 1. Do not reset those systems as a proxy recovery step.
