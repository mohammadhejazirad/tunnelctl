# Keyboard menu

On tested Linux amd64, `tunnelctl` without arguments opens the menu when both
stdin and stdout are terminals. `tunnelctl tui` opens the same interface. Use
`--core /absolute/protected/sing-box` and `--data-dir /private/path` as needed.
Redirected input/output, `--json`, `--non-interactive` and `TERM=dumb` never enter
a blocking menu: no-argument startup returns help (or a JSON usage result), while
explicit `tui` returns an unsupported-capability error. Direct CLI remains independent.

Main menu: Overview, Connect, Profiles, Subscriptions, Sessions, Logs, Diagnostics,
Settings, Help, Exit. TUN, service management and Xray are unavailable, not active
dummy controls. Bubble Tea operations call the same application services as CLI;
network operations run asynchronously, not from terminal rendering.

## Navigation

- Arrows or j/k move; Enter opens/selects; Esc returns to the main menu.
- `?` opens Help; `q` requests exit; `s` opens Connect; `x` confirms stop.
- 1–4 and Tab are shortcuts for Overview, Profiles, Subscriptions and Logs.
- Forms: Enter advances/saves, Tab/up/down chooses fields, Ctrl+U clears the field,
  Backspace edits, Esc/Ctrl+C discards. Paste is supported with bounded field sizes.
- During operations, Esc/Ctrl+C cancels and waits for completion/cleanup. `q`
  cancels pending work, waits, then checks the session and offers exit options.

No mouse, emoji or special font is needed. Resize scrolls bounded list windows;
very small terminals show a size warning. `NO_COLOR`, `--no-color`, `--plain`
disable authored colors; terminal rendering still uses normal terminal control
codes. Untrusted ANSI/OSC, bidi and control sequences are removed from names/text.

## Profiles

Open Profiles, use `i` for hidden link entry or `f` for an owned 0600 regular
input file. Supported links are precisely the CLI's Shadowsocks SIP002/TCP subset.
Hidden input is never rendered, including pasted text. Import does not connect.
Use `/` to search names; Enter opens redacted details and actions: Select,
Validate generated config with approved core, Rename, Remove. `r`/`d` are list
shortcuts. Removing an active profile is rejected by the shared service.

## Subscriptions

Open Subscriptions and `a`: name, hidden HTTPS URL, optional explicit narrow
private CIDRs and private PEM CA path. No fetch occurs during Add. Never disable
TLS validation to fix trust failures. Enter opens safe metadata/actions; `r`
renames and `d` confirms removal. Removing a source retains standalone profiles.

`p`/`u` fetches a preview asynchronously. Review added/removed/unchanged counts;
Enter/y approves the snapshot, Esc/n cancels. The private preview token is used
internally but not displayed. Apply refetches and verifies that token; if the
source changed or a lock/conflict/fetch/parse fails, nothing staged is committed.
No automatic active-session switch occurs. A conditional 304 retains healthy data.
Logs never display URLs, validators, credentials, response bodies or core output.

## Connect and lifetime

Only Proxy is selectable. Choose/select a profile in Profiles, then configure an
unprivileged loopback port, optional duration (e.g. 10m) and optional private
listener authentication file. Settings opens a persistent multifield draft shared
with CLI: port, duration, auth path, hidden explicit probe target, core path, timeout
and color. Tab/arrows move, Enter advances to Save, Enter on Save commits all;
Esc discards all. A stale snapshot fails with a reload remedy. Success appears
only after durable commit. An active listener/core/deadline is unchanged; reconnect
explicitly. CLI core/timeout/style overrides remain invocation-only. See storage.md.
Profile details also offers atomic tags/favorite editing.

Connect requires confirmation and starts the existing **detached** supervisor.
Duration is enforced independently of the UI. No UI-owned foreground session is
advertised. The menu stays unprivileged. Listener readiness is not verified
upstream connectivity. Overview separately shows Disconnected, Connecting,
Listener ready (upstream unverified), or a successful explicit probe.
Traffic and latency remain Unavailable.

An optional, explicitly configured HTTPS destination can be probed once after
confirmation. Credentials/query/fragment targets are refused; redirects are not
followed, TLS stays verified, and at most 4 KiB of response body is read with a
10-second deadline. A passing probe is local to this menu/session, not written
as a persistent claim in `status`. A timeout cannot prove bad upstream credentials.

Exit refreshes real session state. For a running detached session choose Leave
running and exit, Disconnect confirmed session and exit, or Return. Stop is bound
to the confirmed authenticated session ID; a replacement session is not stopped
by stale confirmation. Terminal loss cancels in-flight work and waits for owned
startup cleanup; an established detached session remains running. Use `status`
and `stop` from the same user/storage after reopening the terminal.

## Logs and diagnostics

Logs show at most 256 last-session state events with bounded rendering, never raw
core logs. Diagnostics checks private storage, approved core version/artifact and
capabilities without an unsolicited upstream request. Errors stay in the menu
with a sanitized remedy. CLI doctor/status/stop remain recovery paths.

Evidence: deterministic model/form tests, controlled TLS preview/commit/cancel
tests, Linux PTY explicit/automatic startup, real-core menu connection, detached
exit and session-bound stop. Not a complete screen-reader/visual/native-platform
certification. Privileged TUN has no suitable isolated acceptance environment.
