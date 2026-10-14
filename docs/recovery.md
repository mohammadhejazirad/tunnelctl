# Offline recovery

Normal stop: press Ctrl+C in the foreground terminal, or run `tunnelctl stop`
from another terminal as the same user with the same --data-dir (if overridden).
Repeated stop is safe. SIGTERM/SIGHUP behave like Ctrl+C. Closing a terminal only
delivers this behavior if the terminal/shell actually sends SIGHUP. Explicit `start`
detaches into a new session: terminal closure and TUI quit leave it running. Use
duration or `stop`. Boot services and arbitrary nohup lifecycles are unsupported.

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
restore only a known good schema compatible with your binary. Schema-1 migration
preserves profiles.schema1.backup.json; schema-2 migration preserves
profiles.schema2.backup.json. Backups include credentials and subscription URLs.
A staging file is not automatically valid. No repair overwrites malformed data.
See versioning.md before attempting a downgrade.
See storage.md for protected backup, corrupt/newer-schema refusal, defaults reset
versus explicit data deletion, and recovery after an interrupted atomic write.

There are no routes, DNS settings, firewall rules, services or shell configuration
to roll back in this proxy release. Do not reset them as a proxy recovery step.
