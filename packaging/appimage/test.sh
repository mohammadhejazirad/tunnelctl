#!/usr/bin/env bash
set -euo pipefail
assets=${1:?Usage: test.sh ASSETS TAG VERIFIED_CORE}
tag=${2:?}
core=${3:?}
assets=$(cd "$assets" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
image="$assets/tunnelctl_${tag#v}_linux_amd64.AppImage"
# Stable extraction is the supported no-FUSE session mode. Transient runtime
# sessions are separately tested for explicit refusal, not silently accepted.
(cd "$stage" && "$image" --appimage-extract >/dev/null)
TUNNELCTL_TEST_BIN="$stage/squashfs-root/usr/bin/tunnelctl" TUNNELCTL_TEST_CORE="$core" go test -count=1 -run 'TestRealProxyMethodsAndSafeStop|TestPortCollisionDurationAndSignals|TestCoreCrashAndSupervisorDeath|TestSOCKSHostnameAndVerifiedTLS|TestDetachedExec' ./test/integration
