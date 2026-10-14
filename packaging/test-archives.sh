#!/usr/bin/env bash
set -euo pipefail
assets=${1:?Usage: test-archives.sh ASSETS TAG}
tag=${2:?}
version=${tag#v}
assets=$(cd "$assets" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64; do
  ext=tar.gz
  if [[ "$target" == windows_amd64 ]]; then ext=zip; fi
  name="tunnelctl_${version}_${target}.${ext}"
  cp "$assets/$name" "$stage/$name"
  (cd "$stage" && sha256sum "$name") >> "$stage/checksums.txt"
done
TUNNELCTL_RELEASE_DIR="$stage" TUNNELCTL_RELEASE_TAG="$tag" go test -count=1 -v ./test/release
