# Compatibility status

Machine-readable records are in ../compatibility.json. Application 0.1.0, schema 1,
Go 1.27.2, Cobra 1.10.1 and sing-box 1.14.3 are pinned. Go transitive dependencies
are locked in go.mod/go.sum. Bubble Tea, Lip Gloss and Xray are deferred, not
installed unused dependencies.

Linux amd64 proxy was tested on Ubuntu 26.04.1 LTS, kernel 7.0.0-38-generic,
UID 1000, local Linux filesystems and procfs with permitted loopback sockets.
No systemd is used. A local filesystem supporting Unix permissions, atomic rename,
fsync and flock is required. procfs and SO_PEERCRED are required for the Linux
lifecycle implementation. The minimum supported kernel and broader distribution
matrix have not been established.

The current source workspace is on a fuseblk mount that presents nobody ownership
and 0777 modes. It is usable for source/build artifacts but was rejected for the
core and private data. Tests use private /tmp directories with real Unix permissions.

Runtime commands explicitly reject non-Linux platforms; core approval is currently
linux-amd64 only. macOS/Windows builds do not implement native private storage or
session ownership. Version/help remain usable in compiled binaries. Linux arm64
requires an independently approved artifact and native tests before support.
PowerShell, macOS, containers, WSL and other distributions are not functionally
certified. TUN is unsupported everywhere in this phase.
