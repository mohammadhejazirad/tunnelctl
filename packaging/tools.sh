#!/usr/bin/env bash
# Pinned official tools only; no downloaded script is evaluated.
set -euo pipefail
if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  echo 'The pinned packaging bootstrap requires a Linux amd64 build host. Native MSI/pkg jobs use their own target tools.' >&2
  exit 3
fi
tools_dir=${1:?Usage: tools.sh NEW_TOOL_DIRECTORY}
mkdir -p "$tools_dir"
tools_dir=$(cd "$tools_dir" && pwd)
fetch() {
  curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 --retry 3 --connect-timeout 20 --max-time 240 "$1" --output "$tools_dir/$2"
  printf '%s  %s\n' "$3" "$tools_dir/$2" | sha256sum --check
}
fetch https://github.com/goreleaser/goreleaser/releases/download/v2.18.3/goreleaser_Linux_x86_64.tar.gz goreleaser.tar.gz b8dee4728bca1aba0d71a832a17cd0a3799f75372878e7ffa36aef2343bce0bc
tar -xzf "$tools_dir/goreleaser.tar.gz" -C "$tools_dir" goreleaser
fetch https://github.com/AppImage/appimagetool/releases/download/1.9.1/appimagetool-x86_64.AppImage appimagetool ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0
# The upstream runtime has only a moving URL. Digests pin the bytes to upstream
# commit 8f39b89e2ac31e1640b3d3f7e9a5108e6ce805fa; changed bytes FAIL CLOSED.
fetch https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64 runtime-x86_64 156f4bdbde9c52d01814600013e0a273f0118dc2de98975f3c8c63427ec79074
fetch https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-aarch64 runtime-aarch64 b4ff0030242d0c3bb12ce40541828303cf167493f4793456f0436edd6255c39d
# Corresponding runtime/libfuse source (including runtime's fuse patch/build
# scripts) and full static-library notices accompany each distributed AppImage.
fetch https://codeload.github.com/AppImage/type2-runtime/tar.gz/8f39b89e2ac31e1640b3d3f7e9a5108e6ce805fa type2-runtime-source.tar.gz c449eb42c2190820872631f5e22f8e15b1e985a36dc4a9e2de3585bb7536c785
fetch https://github.com/libfuse/libfuse/releases/download/fuse-3.15.0/fuse-3.15.0.tar.xz fuse-3.15.0.tar.xz 70589cfd5e1cff7ccd6ac91c86c01be340b227285c5e200baa284e401eea2ca0
fetch https://raw.githubusercontent.com/libfuse/libfuse/fuse-3.15.0/LGPL2.txt libfuse-LGPL2.txt dc626520dcd53a22f727af3ee42c770e56c97a64fe3adb063799d8ab032fe551
fetch 'https://git.musl-libc.org/cgit/musl/plain/COPYRIGHT?h=v1.2.5' musl-COPYRIGHT f9bc4423732350eb0b3f7ed7e91d530298476f8fec0c6c427a1c04ade22655af
fetch https://raw.githubusercontent.com/vasi/squashfuse/0.5.2/LICENSE squashfuse-LICENSE 9e909cc8a8ba27b1a649c964cf9eda37de911f3138c2b64e6d2f01703904ac13
fetch https://raw.githubusercontent.com/facebook/zstd/v1.5.6/LICENSE zstd-LICENSE 7055266497633c9025b777c78eb7235af13922117480ed5c674677adc381c9d8
fetch https://raw.githubusercontent.com/madler/zlib/v1.3.1/LICENSE zlib-LICENSE 845efc77857d485d91fb3e0b884aaa929368c717ae8186b66fe1ed2495753243
chmod 0755 "$tools_dir/goreleaser" "$tools_dir/appimagetool"
(cd "$tools_dir" && ./appimagetool --appimage-extract >/dev/null)
