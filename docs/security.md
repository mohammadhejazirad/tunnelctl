# Security boundary

Profiles, stdin, files and share-link fields are untrusted. One URI is bounded at
64 KiB, decoded credentials at 8 KiB, passwords at 4096 UTF-8 bytes, names at 256
bytes, storage at 10 MiB and 10,000 profiles. Scheme, endpoint, cipher and fields
are validated; plugins, all query parameters, paths beyond an optional slash,
legacy whole-link Base64 and other protocols are rejected. Standard/URL-safe,
padded/unpadded strict Base64 userinfo and percent-encoded credentials are supported.
IPv6 literals are accepted; scoped addresses, ambiguous structural data, invalid
hostname labels and unsupported security/transport extensions are refused.

ANSI/OSC sequences including payloads, C0/C1, formatting/bidi controls and invalid
name encodings cannot reach terminal output. Credentials never appear in public
Profile JSON, errors or help. The core config is internal and never exported.
Core logs are disabled and subprocess output is suppressed except for a bounded
version probe. No telemetry, health-check destination or external-IP call is made
automatically. JSON failures use sanitized English messages and stable codes.

Shadowsocks AEAD provides its protocol's confidentiality; it is not TLS and does
not add a new transport/security protocol. HTTPS used through the proxy retains
the client's TLS certificate checks. No insecure TLS flag exists. TCP traffic
explicitly directed to the proxy uses the upstream; unrelated host traffic is
unaffected. Endpoint hostnames bootstrap through the core's default local resolver
in this tested configuration. Host DNS is not changed, and no universal DNS-leak,
UDP/ICMP or full-tunnel protection claim is made.

Default loopback access is unauthenticated and therefore available to other local
users. Listener authentication from a private file is supported and tested. The
current UID, root, malware with that user's privileges, hostile kernels and physical
disk attackers can access secrets. File permissions are not encryption at rest.
Filesystems with weak modes/locking or hostile path ancestors are not certified.

The core is pinned by version and executable SHA-256. Descriptor-based execution
protects against pathname replacement, not same-UID in-place modification. The
proxy rejects set-ID or file-capability executables; it never needs an elevated
core or persistent capabilities. Unknown capability inspection errors fail closed.
The official artifact digest and download were obtained through upstream HTTPS; a
checksum from the same origin is integrity evidence, not independent authenticity
or a supply-chain audit. CI requests release provenance; local dry-runs have no
attestation. Archives include dependency license texts and do not bundle the core.
Configs are unlinked before validation/run and close on normal cleanup or
process death. Memory, swap and open procfs descriptors are not encrypted/zeroized.

Session locks serialize one runtime root; separate roots may host independent
proxy sessions, subject to normal port conflicts. No host-wide TUN ownership is
implied. Only owned child handles receive signals. Graceful shutdown escalates
after three seconds to forced termination, followed by a three-second wait. Cleanup
errors are explicit (code 7); unexpected runtime file types are left for inspection.

This implementation has not had an independent security audit. Power loss,
filesystem faults at every transaction boundary, multiple local users, arbitrary
hostile IPC clients and packet-level IPv6/DNS leak scenarios are not certified.

Subscription URLs/HTTP validators are private schema-3 data, never public output.
Preview tokens use HMAC with a private random per-source key, not public plaintext
hashes of password-bearing feeds. Key and validators never enter public metadata.
HTTPS fetch enforces bounds, certificate verification, origin-limited redirects
and literal-address dialing after validating every DNS answer. Private CIDRs are
explicit exceptions, not a TLS bypass. See subscriptions.md for transaction and
durability boundaries. State logs contain only validated state names, timestamps
and session IDs, not core output or traffic. The narrow repository hygiene scanner
does not replace a complete secret scan or human review.
