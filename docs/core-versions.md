# Pinned core and toolchain

Verified against official upstream metadata/source on 2026-10-14:

- Go 1.27.2: https://go.dev/dl/ ; archive https://go.dev/dl/go1.27.2.linux-amd64.tar.gz
- Cobra 1.10.1: https://github.com/spf13/cobra/tree/v1.10.1 (Go requirement 1.15)
- Bubble Tea v2.1.0: https://github.com/charmbracelet/bubbletea/tree/v2.1.0
  (`charm.land/bubbletea/v2`, Go requirement 1.26); official KeyPressMsg/NewView API.
- Lip Gloss v2.0.6: https://github.com/charmbracelet/lipgloss/tree/v2.0.6
  (`charm.land/lipgloss/v2`, Go requirement 1.25).
- sing-box 1.14.3: https://github.com/SagerNet/sing-box/releases/tag/v1.14.3

Core archive:
https://github.com/SagerNet/sing-box/releases/download/v1.14.3/sing-box-1.14.3-linux-amd64.tar.gz

```text
Archive SHA-256: e2bdf179c15a3652955dc44867e33ca0965c9c9b91b34267dcc5a5d639a5feee
Executable SHA-256: 768d31fa7f68debc067bb4626539498a1e8751d36a7b99e1a54bee6a23815f93
Go archive SHA-256: ecbadb99091a3f46e31f5f934b068b1864eafa7995211b39eaddf76996045fe5
```

Download as a normal user into a private directory. Verify the archive before
extracting using `sha256sum`, inspect archive paths/types, extract only under that
directory, then verify the executable hash. No downloaded script is necessary.
The command path must be absolute and the executable not group/world writable.
Preserve the upstream license. `core verify` checks executable digest/version;
doctor additionally checks private storage. Neither connects to an upstream.

The adapter was checked against the exact-tag Shadowsocks, VLESS, VMess, Trojan,
TLS/Reality and V2Ray transport option sources plus `option/simple.go`,
`option/route.go`, `cmd/sing-box/cmd_check.go` and dial/DNS documentation. Its
emitted configs were accepted by the real core; SS, VLESS, VMess and Trojan have
controlled real-request tests. It uses
`check -c PATH`, `run -c PATH` and `version`. Core build tags/revision are in
compatibility.json. The archive includes libcronet.so; tested proxy paths do not
call Cronet features. No unlisted protocol/transport capability is certified.

No runtime latest selection, automatic install or update is implemented. New core
versions require schema/API review, integrity records and real acceptance tests,
then a deliberate adapter/manifest change. Same-origin checksums do not establish
independent artifact authenticity; signatures/provenance review remains a release gate.
