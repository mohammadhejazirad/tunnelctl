# Explicit private sample testing (development only)

The application, normal `go test ./...`, CI and release packaging never open a
root sample file or contact its endpoints automatically. The separate Go tool
`cmd/private-test` is opt-in and is **not included in release archives**. Public
fixtures are synthetic; a controlled real-core test validates the harness itself.

## Before use

Use exactly `.gitignore`, with `/links_and_subs.txt`, not a misspelled file.
Check metadata only:

```sh
git check-ignore -v --no-index links_and_subs.txt
git ls-files --error-unmatch links_and_subs.txt  # Must report not tracked.
git log --all --format='%h' -- links_and_subs.txt  # Investigate any matching commits.
```

If tracked, untrack without deleting the local input. Prior commits retain secrets:
revocation and authorized history remediation are separate actions. Do not rewrite
history implicitly. The harness refuses tracked/nonignored in-repository inputs.
An explicit regular file outside the repository is also supported; symlinks/devices
are refused. Never move, overwrite or remove a maintainer's input during testing.

The source drive might not implement Unix permissions. Only the development
harness may read an explicitly selected sample input there; normal profile/store
permissions remain strict. All derived state is created in a fresh 0700 temporary
directory on a permission-supporting filesystem, with 0600 secret files. Generated
configs are unlinked immediately. Cleanup is checked and reported.

## Run

File syntax: one URL/share link per nonempty line. Blank lines and lines beginning
with `#` are ignored. Other annotation lines are reported as invalid input; they
are not guessed or executed. Bounds: 5 MiB file, 64 KiB line; at most 20 chosen
samples, 20 tested profiles per source and five total sequential connection probes.
Concurrency is one. Download/decode/parser bounds and SSRF policies remain those
in [subscriptions](subscriptions.md); private CIDR exceptions cannot be enabled
silently by this tool. Only currently supported protocols can be committed/run.

```sh
# Build on a permission-supporting filesystem; no credentials in argv.
go build -o /private/tools/private-test ./cmd/private-test
# Fetch/parse and generated-config checks only; no upstream connection probe.
/private/tools/private-test --file /private/links_and_subs.txt \
  --core /absolute/protected/sing-box --max-samples 5 --max-profiles 2
# Explicitly authorize at most one small request, only if a supported profile exists.
/private/tools/private-test --file /private/links_and_subs.txt \
  --core /absolute/protected/sing-box --connect --probe-url https://example.com/ \
  --max-samples 5 --max-profiles 2 --max-connections 1 --timeout 30s
```

The probe target is configurable, explicitly disclosed, TLS-verified and bounded
to a 4-KiB body/10-second request. No speed test, third-party feed conversion,
analytics or concurrent scan occurs. Total per-sample deadline includes staging
and refetches; default 30 seconds, allowed 1–60 seconds. HTTPS feed fetch has its
own shorter limits, same-origin redirects and pinned resolved-address dialing.

Healthy subscriptions undergo fetch, parse, preview, atomic commit, manual refetch
and stable identity/custom-label/selection checks. At most five direct fetches
per healthy source occur (initial fetch, preview/apply, refetch preview/apply);
304 reduces this. Empty/malformed feeds cannot delete last-good state. Malicious
responses, concurrency and failure scenarios belong to synthetic controlled tests,
never pressure-testing or altering private real services.

JSON includes only anonymous IDs such as sample-001, input kind, stage result,
counts and evidence-based error category. No source name, host, URL, ID, token,
response, validator or raw error text is serialized. Sample failure gives
`passed:false` and exit 5; harness/setup/cleanup failures also return nonzero.
Skipped limits/absence of optional probes are not connectivity success. Truncation
is explicit. Missing later stages mean **not run**, not passed. TUN is unsupported
and untested in this environment.

Categories distinguish invalid input, unsupported capability, DNS, certificate
TLS validation, HTTP authentication/access denial, policy, timeout/cancellation,
unreachability, local permission/conflict/core problems and unknown failures.
Generic proxy timeout cannot distinguish authentication from upstream outage.
Successful fetch, accepted config, ready listener and real HTTPS success are
separate stages. Never label skipped network stages as connection verified.

Do not save private-derived reports in docs, examples, snapshots or fixtures.
If locally retaining even anonymized output, use private 0600 storage **outside**
the repository. Never upload it to CI/releases. `/private-results/`,
`/private-cache/`, `/.private/`, `*.private.json` and `*.private.log` are ignored
and forbidden by hygiene/package checks, but ignore rules are not a content audit.
Release builder lists public docs/examples explicitly, rejects secret-named files
and symlinks, and packages only staged public artifacts. `.dockerignore` denies
the entire context; no Docker installation is supported. Do not archive the source
tree with a wildcard or copy local reports to an allowlisted public document.

## Controlled public harness test

Build the ordinary CLI and development harness in protected temporary storage,
then set `TUNNELCTL_TEST_BIN`, `TUNNELCTL_TEST_CORE` and
`TUNNELCTL_PRIVATE_TEST_BIN` (paths only) and run `go test ./test/integration`.
The harness acceptance test generates a local Shadowsocks server, a synthetic
TLS HTTPS target and synthetic credentials. Its separate process trusts the
explicit synthetic CA through SSL_CERT_FILE; TLS verification is never disabled.
CI only runs this synthetic path, never a maintainer's real samples.
