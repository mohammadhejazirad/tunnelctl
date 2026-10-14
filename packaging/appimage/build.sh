#!/usr/bin/env bash
set -euo pipefail
work=${1:?Usage: build.sh PACKAGE_WORK TOOL_DIRECTORY VERSION}
tools_dir=${2:?}
version=${3:?}
work=$(cd "$work" && pwd)
tools_dir=$(cd "$tools_dir" && pwd)
for arch in amd64 arm64; do
  runtime_arch=x86_64
  if [[ "$arch" == arm64 ]]; then runtime_arch=aarch64; fi
  appdir="$work/AppDir-$arch"
  mkdir -p "$appdir/usr/bin" "$appdir/usr/share/doc/tunnelctl"
  install -m755 "$work/goreleaser/app_linux_${arch}"*/tunnelctl "$appdir/usr/bin/tunnelctl"
  cp -R "$work/public/." "$appdir/usr/share/doc/tunnelctl/"
  runtime_docs="$appdir/usr/share/doc/tunnelctl/APPIMAGE_RUNTIME"
  mkdir "$runtime_docs"
  for file in type2-runtime-source.tar.gz fuse-3.15.0.tar.xz libfuse-LGPL2.txt musl-COPYRIGHT squashfuse-LICENSE zstd-LICENSE zlib-LICENSE; do
    install -m644 "$tools_dir/$file" "$runtime_docs/$file"
  done
  install -m644 packaging/appimage/LICENSE.runtime "$runtime_docs/LICENSE.runtime"
  install -m644 packaging/appimage/RUNTIME-NOTICE.md "$runtime_docs/README.md"
  # A relative symlink, not a terminal-emulator script: arguments, streams,
  # exit status and signals go directly to the actual CLI process.
  ln -s usr/bin/tunnelctl "$appdir/AppRun"
  install -m644 packaging/appimage/tunnelctl.desktop "$appdir/tunnelctl.desktop"
  install -m644 packaging/appimage/tunnelctl.svg "$appdir/tunnelctl.svg"
  ARCH="$runtime_arch" VERSION="$version" "$tools_dir/squashfs-root/AppRun" \
    --no-appstream --runtime-file "$tools_dir/runtime-$runtime_arch" \
    "$appdir" "$work/assets/tunnelctl_${version}_linux_${arch}.AppImage"
done
