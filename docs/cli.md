# CLI contract, schema 1

Implemented commands: help, version, init, profile import/list/show/select/rename/
remove; subscription add/list/show/rename/remove/update; run, start, exec, status,
stop, logs, doctor, core list/verify, tui and completion. Profile identifiers are full random IDs or unique
sanitized names; duplicate names require IDs. Duplicate connection identities are
rejected, and the first import becomes selected. Imported state remains distinct
from successful connectivity.

Import requires exactly one of `--file PATH` or `--stdin`. Files must be private
regular files owned by the current UID. Show always redacts credentials, including
when `--redacted=false` is passed: secret export is not available.

Global arguments: `--data-dir`, `--core`, `--json`, `--timeout`, `--plain`,
`--no-color`, `--non-interactive`. CLI output is always plain; styling flags affect
TUI. NO_COLOR disables color; TERM=dumb, nonterminal input/output, JSON and
non-interactive refuse TUI. `--config` and `--verbose` are
not implemented. Core path defaults to TUNNELCTL_CORE or /usr/local/bin/sing-box
and must be absolute. Timeout is greater than zero and at most two minutes.

Run flags: `--profile`, `--mode proxy`, `--duration 10m`, `--listen-port 2080`,
`--proxy-auth-file`. Negative durations are invalid; zero has no deadline. Only
ports 1024–65535 and fixed IPv4 loopback are supported. Foreground JSON run emits
exactly one final result after shutdown; use a separate `status --json` while it
runs. Plain mode prints readiness and the final cleanup result. A failed command
never prints a successful JSON envelope. Help is ordinary English text.

All successful machine results use:

```json
{"schema_version":1,"ok":true,"data":{"state":"idle","cleanup":"complete"}}
```

Errors use `ok: false`, `data: null`, and `error` containing `code`, `message`,
`remediation`. Diagnostics go to stderr in plain mode; JSON errors go to stdout.
Use `--json` before other arguments when parsing potentially invalid flags.
Underlying error strings and user arguments are not echoed.

`start` supports profile, duration, listener port and private authentication file,
and returns after real listener readiness. Mode is proxy only; there is no dummy
TUN flag. `exec -- COMMAND ARGS` launches directly, without constructing a shell.
It preserves child exit status unless cleanup fails; failed-child JSON includes
data.child_exit_code (the exception to data:null). JSON child output is sent to
stderr. Logs are bounded last-session state events; no follow or raw core logs.

Subscription add requires --name and --url-stdin; no secret URL argument exists.
Update supports --preview, --accept-removals and --if-preview TOKEN. See
subscriptions.md for transaction, identity, source-removal and active-session rules.
TUI confirmation only authorizes the same shared use cases, not extra capabilities.
Completion supports bash/zsh/fish/powershell and never installs shell changes.

| Exit | Meaning |
|---|---|
| 0 | Successful operation and cleanup |
| 1 | Unexpected failure |
| 2 | Invalid input |
| 3 | Unsupported protocol/core/schema/platform capability |
| 4 | Private storage or permission requirement |
| 5 | Core/network failure or readiness deadline |
| 6 | Concurrent session, unavailable port or ambiguous state |
| 7 | Cleanup incomplete; takes precedence over another failure |

Status reports process_running, listener_ready, connection_tested, start/deadline,
profile ID, core version and listener. `connection_tested` is false because this
version does not perform a runtime upstream probe. Statistics are Unavailable.
Status has no credentials, persisted PID authority or invented traffic counters.
