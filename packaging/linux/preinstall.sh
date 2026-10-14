#!/bin/sh
set -eu
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH
# Refuse unowned files, including symlinks/dangling symlinks. Normal upgrades
# are allowed only when this same package manager already owns the binary.
if [ -e /usr/bin/tunnelctl ] || [ -L /usr/bin/tunnelctl ]; then
    # The first successful ownership query is authoritative. An unrelated
    # package-manager tool may also be installed; its failed query must not
    # erase the actual owner's answer and break upgrades.
    if command -v dpkg-query >/dev/null 2>&1 && owner=$(dpkg-query -S /usr/bin/tunnelctl 2>/dev/null); then :
    elif command -v rpm >/dev/null 2>&1 && owner=$(rpm -qf --qf '%{NAME}' /usr/bin/tunnelctl 2>/dev/null); then :
    elif command -v apk >/dev/null 2>&1 && owner=$(apk info --who-owns /usr/bin/tunnelctl 2>/dev/null); then :
    elif command -v pacman >/dev/null 2>&1 && owner=$(pacman -Qoq /usr/bin/tunnelctl 2>/dev/null); then :
    else owner=''
    fi
    case "$owner" in
        tunnelctl|"tunnelctl: /usr/bin/tunnelctl"|"/usr/bin/tunnelctl is owned by tunnelctl-"*) ;;
        *) echo 'Refusing to replace an unowned or differently owned /usr/bin/tunnelctl. Remove or migrate the other installation explicitly.' >&2; exit 1 ;;
    esac
fi
