#!/bin/bash
set -euo pipefail
assets=${1:?Usage: test.sh ASSETS TAG ARCH}
tag=${2:?}
arch=${3:?}
version=${tag#v}
assets=$(cd "$assets" && pwd)
pkg="$assets/tunnelctl_${version}_darwin_${arch}.pkg"
test ! -e /usr/local/bin/tunnelctl
test ! -e '/Library/Application Support/tunnelctl'
canary_dir=$(mktemp -d "$HOME/Library/Application Support/tunnelctl-package-test.XXXXXXXX")
printf '%s\n' synthetic-preserved-data > "$canary_dir/preserved.txt"
# Unmanaged-path refusal, including Homebrew-like symlinks.
sudo mkdir -p /usr/local/bin
sudo ln -s /nonexistent-synthetic-collision /usr/local/bin/tunnelctl
if sudo installer -pkg "$pkg" -target /; then echo 'Collision not refused' >&2; exit 1; fi
test "$(readlink /usr/local/bin/tunnelctl)" = /nonexistent-synthetic-collision
sudo unlink /usr/local/bin/tunnelctl
sudo installer -pkg "$pkg" -target /
cd /tmp
result=$(/usr/local/bin/tunnelctl version --json)
printf '%s\n' "$result" | /usr/bin/plutil -extract data.tag raw -o - - | grep -Fx "$tag"
tunnelctl --help
test "$(readlink /usr/local/bin/tunnelctl)" = '/Library/Application Support/tunnelctl/tunnelctl'
test "$(stat -f %u '/Library/Application Support/tunnelctl/uninstall.sh')" = 0
test "$(stat -f %Lp '/Library/Application Support/tunnelctl/uninstall.sh')" = 755
sudo installer -pkg "$pkg" -target / # Reinstall/repair, not an old-public-release upgrade.
sudo '/Library/Application Support/tunnelctl/uninstall.sh'
test ! -e /usr/local/bin/tunnelctl
test ! -e '/Library/Application Support/tunnelctl/tunnelctl'
test "$(cat "$canary_dir/preserved.txt")" = synthetic-preserved-data
if pkgutil --pkg-info io.github.mohammadhejazirad.tunnelctl; then exit 1; fi
echo 'Native pkg collision/install/reinstall/PATH/uninstall passed. Proxy runtime remains unsupported.'
