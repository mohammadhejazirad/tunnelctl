# Transactional subscriptions

Formats: UTF-8 newline `ss://`, `vless://`, `vmess://` and `trojan://` links; the
same payload in strict standard or URL-safe Base64, padded/unpadded. No plugins,
insecure TLS bypass, arbitrary JSON/YAML core configs or legacy whole-link
Shadowsocks encoding. One malformed line rejects the entire update.

Limits: 5 MiB wire, 10 MiB expanded/decoded, 10,000 nonempty links (duplicates count),
64 KiB per link, 20-second deadline, 3 same-origin HTTPS redirects, bounded headers
and validators. Gzip is subject to both limits. Ctrl+C/timeouts preserve the healthy
snapshot when canceled before commit.

Direct unprivileged HTTPS, TLS >=1.2 and verified certificates; no environment proxy
or converter. Every DNS answer must pass the IP policy. Dial the validated literal
without another lookup, then verify the TCP peer. Private/loopback/link-local,
metadata, CGNAT, reserved and transition/NAT64 destinations are blocked by default.
Redirects cannot downgrade, leave origin or add/change URL credentials.
ETag/Last-Modified remain private and commit only with healthy data. 304 requires a
previous healthy snapshot.

Trusted private source: explicitly add a narrow canonical CIDR, for example
`--allow-private-cidr 192.168.50.10/32`. A controlled CA uses an owned `0600` PEM
`--ca-file`; keep it available for refresh. No TLS bypass. URLs, validators and CA
paths are not public metadata; approved CIDRs are intentional nonsecret policy.

Fetch -> bounded decode -> strict parse/validate -> identity/diff -> atomic store
replacement. Sources and profiles share one authoritative JSON file. Per-source
flock serializes fetches; store flock and source-revision CAS protect against
concurrent rename/removal. Conflicts return code 6, not silent overwrites. Failure
before commit preserves the last healthy file. Fsync failure after rename reports
code 7: replacement may already be visible and is not claimed to be rolled back.

Preview changes neither data nor validators. Its HMAC token binds source identity/
name, revision and fetched body using a random 128-bit private per-source key. A
public unsalted hash of credential-bearing feed data would permit dictionary
checking; that is not used. Commit refetches; `--if-preview TOKEN` rejects changed
content. Unselected removals require `--accept-removals`; empty updates always fail.
Selected removed profiles become standalone and stay selected. IDs/timestamps and
existing labels survive retained connection identities. Credential/endpoint changes
create new identities. User labels do not determine deduplication.

Removing a subscription retains all profiles as standalone; profile deletion is
separate. Old URLs may remain in private backups/staging: deletion is not secure
disk erasure. Updates never replace/disconnect active sessions. No scheduled or
hidden connectivity requests. Atomicity covers one source update, not a distributed
transaction across multiple sources.
