#!/bin/bash
# Disposable Arch container only. Network is used for the build prerequisites,
# never for private feeds or a release download (the tarball is locally cached).
set -euo pipefail
version=${1:?Usage: test-pkgbuild.sh VERSION}
pacman -Syu --noconfirm --needed base-devel
pacman -Q pacman fakeroot binutils
useradd -m builder
install -d -o builder -g builder /tmp/arch-build
install -m644 "/assets/tunnelctl_${version}_linux_amd64.PKGBUILD" /tmp/arch-build/PKGBUILD
install -m644 "/assets/tunnelctl_${version}_linux_amd64.tar.gz" "/tmp/arch-build/tunnelctl_${version}_linux_amd64.tar.gz"
su builder -c 'cd /tmp/arch-build && makepkg --noconfirm'
mapfile -t packages < <(find /tmp/arch-build -maxdepth 1 -name '*.pkg.tar.zst' -type f)
test "${#packages[@]}" = 1
# The base Docker image excludes docs/licenses globally; override its NoExtract
# only in this test invocation, not in any distributed installer configuration.
sed '/^[[:space:]]*NoExtract[[:space:]]*=/d' /etc/pacman.conf > /tmp/package-pacman.conf
pacman --config /tmp/package-pacman.conf -U --noconfirm "${packages[0]}"
cd /tmp
tunnelctl version --json | grep -F "\"application\":\"$version\""
tunnelctl --help >/dev/null
test -s /usr/share/licenses/tunnelctl/LICENSE
test -s /usr/share/doc/tunnelctl/docs/recovery.md
pacman --config /tmp/package-pacman.conf -R --noconfirm tunnelctl
test ! -e /usr/bin/tunnelctl
