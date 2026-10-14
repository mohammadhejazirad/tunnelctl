# AppImage runtime, source and modification

This directory concerns only the unmodified AppImage type2 runtime, not the Go
client or any VPN core. It retains the upstream MIT notice and full notices for
its statically linked musl, libfuse (LGPL 2.1), squashfuse, zstd and zlib code.
The app itself is Apache-2.0; no sing-box/Xray binary is included.

Pinned runtime commit: 8f39b89e2ac31e1640b3d3f7e9a5108e6ce805fa.
The accompanying type2-runtime-source.tar.gz includes runtime source, Makefile,
BUILD.md, build/dependency scripts and the libfuse patch. The accompanying
fuse-3.15.0.tar.xz is the complete upstream library source used by that commit's
dependency script. Read LGPL2.txt, including modification/relinking rights.
No restriction on reverse engineering for debugging modifications is imposed.

To modify/relink the runtime, extract those source archives into a separate
owned development directory, review BUILD.md and build in a disposable container
(not the host chroot recipe). Apply patches/libfuse/mount.c.diff to the libfuse
source as described by scripts/common/install-dependencies.sh, substitute your
modified library and build the runtime. Keep the build scripts under review;
some other upstream build dependencies use Alpine packages and a network.
These source materials are for modification, not a byte-reproducibility claim.
Then extract this AppImage using --appimage-extract and repack that AppDir with
appimagetool --runtime-file YOUR_MODIFIED_RUNTIME. No secret data is in AppDir.
Stable extracted AppRun also runs without the AppImage runtime altogether.

The pinned upstream libfuse/squashfuse releases are identifiable in the included
dependency script; other library notices are preserved from their compatible
upstream releases. Sources and notices are checksum-pinned during packaging.
Source origin: https://github.com/AppImage/type2-runtime and
https://github.com/libfuse/libfuse/releases/tag/fuse-3.15.0.
Review redistribution obligations if changing any runtime/dependency; never
silently swap runtime bytes from the moving upstream download URL.
