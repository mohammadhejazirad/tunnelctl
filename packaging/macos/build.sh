#!/bin/bash
set -euo pipefail
assets=${1:?Usage: build.sh ASSET_DIRECTORY TAG ARCH}
tag=${2:?}
arch=${3:?}
version=${tag#v}
assets=$(cd "$assets" && pwd)
stage=$(mktemp -d "${TMPDIR:-/tmp}/tunnelctl-pkg.XXXXXXXX")
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/root/Library/Application Support/tunnelctl" "$stage/scripts"
tar -xzf "$assets/tunnelctl_${version}_darwin_${arch}.tar.gz" -C "$stage/root/Library/Application Support/tunnelctl"
install -m755 packaging/macos/uninstall.sh "$stage/root/Library/Application Support/tunnelctl/uninstall.sh"
install -m755 packaging/macos/preinstall "$stage/scripts/preinstall"
install -m755 packaging/macos/postinstall "$stage/scripts/postinstall"
native=$(go run ./cmd/packaging --mode versions --tag "$tag" | /usr/bin/plutil -extract native_version raw -o - -)
pkgbuild --root "$stage/root" --scripts "$stage/scripts" \
  --identifier io.github.mohammadhejazirad.tunnelctl --version "$native" \
  --install-location / "$stage/component.pkg"
productbuild --package "$stage/component.pkg" "$assets/tunnelctl_${version}_darwin_${arch}.pkg"
pkgutil --expand-full "$assets/tunnelctl_${version}_darwin_${arch}.pkg" "$stage/expanded"
find "$stage/expanded" -name tunnelctl -type f
