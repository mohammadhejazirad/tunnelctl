#!/bin/sh
# Run INSIDE a disposable distro container with /assets read-only.
set -eu
family=${1:?}
version=${2:?}
arch=${3:?}
cd /tmp
cat /etc/os-release
network_before=$(cksum /proc/net/route /etc/resolv.conf)
package="/assets/tunnelctl_${version}_linux_${arch}"
case "$family" in
  deb)
    # Minimal Docker images intentionally exclude docs; include ONLY our docs
    # for this content gate. This does not ship in the package or affect users.
    printf '%s\n' 'path-include=/usr/share/doc/tunnelctl' 'path-include=/usr/share/doc/tunnelctl/*' > /etc/dpkg/dpkg.cfg.d/zz-tunnelctl-package-test
    package="$package.deb"; install_pkg() { dpkg -i "$package"; }; remove_pkg() { dpkg -r tunnelctl; } ;;
  rpm) package="$package.rpm"; install_pkg() { rpm -Uvh --includedocs "$package"; }; remove_pkg() { rpm -e tunnelctl; } ;;
  apk) package="$package.apk"; install_pkg() { apk --no-network add --allow-untrusted "$package"; }; remove_pkg() { apk --no-network del tunnelctl; } ;;
  arch)
    sed '/^[[:space:]]*NoExtract[[:space:]]*=/d' /etc/pacman.conf > /tmp/package-pacman.conf
    package="$package.pkg.tar.zst"; install_pkg() { pacman --config /tmp/package-pacman.conf -U --noconfirm "$package"; }; remove_pkg() { pacman --config /tmp/package-pacman.conf -R --noconfirm tunnelctl; } ;;
  *) exit 2 ;;
esac
mkdir -p /tmp/package-user/config /tmp/package-user/state /tmp/package-user/cache
chmod 700 /tmp/package-user /tmp/package-user/config /tmp/package-user/state /tmp/package-user/cache
printf '%s\n' synthetic-preserved-data > /tmp/package-user/config/canary
chmod 600 /tmp/package-user/config/canary
printf '%s\n' synthetic-unowned-collision > /usr/bin/tunnelctl
chmod 755 /usr/bin/tunnelctl
if [ "$family" = apk ]; then
    # apk cannot reliably abort unpacking after a failed script. Its dedicated
    # hook moves the old file intact to a private, clearly announced backup.
    install_pkg
    backup=$(find /var/backups -name tunnelctl -type f)
    test "$(cat "$backup")" = synthetic-unowned-collision
    test "$(stat -c '%a' "$(dirname "$backup")")" = 700
    remove_pkg
else
    if install_pkg; then echo 'Unowned destination collision not refused' >&2; exit 1; fi
    test "$(cat /usr/bin/tunnelctl)" = synthetic-unowned-collision
    rm /usr/bin/tunnelctl
fi
# dpkg keeps an unconfigured record after a failed preinstall, but no payload.
install_pkg
# Regression: an extra package-manager tool must not erase dpkg ownership.
# This synthetic executable exists only inside these disposable Debian images.
if [ "$family" = deb ]; then
    test ! -e /usr/bin/rpm
    printf '%s\n' '#!/bin/sh' 'echo synthetic-not-owned-by-rpm' 'exit 1' > /usr/bin/rpm
    chmod 755 /usr/bin/rpm
    install_pkg
    rm /usr/bin/rpm
fi
# Fedora 43 merges /usr/sbin into /usr/bin, so PATH can name the same inode
# through /usr/sbin. Assert identity, not a particular textual alias.
test "$(readlink -f "$(command -v tunnelctl)")" = "$(readlink -f /usr/bin/tunnelctl)"
tunnelctl --help >/dev/null
tunnelctl version --json > /tmp/version.json
grep -F "\"application\":\"$version\"" /tmp/version.json
test "$(stat -c '%a' /usr/bin/tunnelctl)" = 755
test "$(stat -c '%u' /usr/bin/tunnelctl)" = 0
test -s /usr/share/doc/tunnelctl/LICENSE
test -s /usr/share/doc/tunnelctl/THIRD-PARTY-NOTICES.txt
test -s /usr/share/doc/tunnelctl/docs/storage.md
for directory in /usr/share/doc/tunnelctl /usr/share/doc/tunnelctl/docs; do
    test "$(stat -c '%a' "$directory")" = 755
    test "$(stat -c '%u' "$directory")" = 0
done
for document in /usr/share/doc/tunnelctl/LICENSE /usr/share/doc/tunnelctl/docs/storage.md; do
    test "$(stat -c '%a' "$document")" = 644
    test "$(stat -c '%u' "$document")" = 0
done
case "$family" in
  deb) dpkg -L tunnelctl > /tmp/package-paths ;;
  rpm) rpm -ql tunnelctl > /tmp/package-paths ;;
  arch) pacman -Qlq tunnelctl > /tmp/package-paths ;;
  apk) apk info -L tunnelctl | sed -n '/^usr\//s,^,/,p' > /tmp/package-paths ;;
esac
while IFS= read -r path; do
    path=${path%/}
    case "$path" in
        ''|/.|/usr|/usr/bin|/usr/share|/usr/share/doc|/usr/bin/tunnelctl|/usr/share/doc/tunnelctl|/usr/share/doc/tunnelctl/*) ;;
        *) echo 'Unexpected package payload location' >&2; exit 1 ;;
    esac
done < /tmp/package-paths
if find /usr/share/doc/tunnelctl -type f \( -name links_and_subs.txt -o -name profiles.json -o -name '*.private.json' -o -name '*.private.log' -o -name session.json -o -name control.sock \) | read -r forbidden; then
    echo 'Private/generated file in package' >&2; exit 1
fi
# Metadata/help and ELF libc independence, not core/runtime certification.
case "$family" in
  deb) dpkg --compare-versions 0.2.401 lt 0.2.999 ;;
  rpm) rpm -q --qf '%{VERSION}\n' tunnelctl ;;
  apk) test "$(apk version -t 0.2.401 0.2.999)" = '<' ;;
  arch) test "$(vercmp 0.2.401 0.2.999)" = -1 ;;
esac
# The application's user data is not created at install time.
test ! -e /root/.config/tunnelctl
remove_pkg
test ! -e /usr/bin/tunnelctl
test "$(cat /tmp/package-user/config/canary)" = synthetic-preserved-data
test "$(cksum /proc/net/route /etc/resolv.conf)" = "$network_before"
if [ "$family" = apk ]; then test "$(cat "$backup")" = synthetic-unowned-collision; fi
echo "Package lifecycle passed: $family $arch; no proxy/TUN certification implied."
