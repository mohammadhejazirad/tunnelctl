#!/bin/sh
set -eu
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH
# apk reports failed pre-install scripts but can still unpack the payload.
# Never rely on `exit 1` alone to protect an unmanaged destination.
if [ -e /usr/bin/tunnelctl ] || [ -L /usr/bin/tunnelctl ]; then
    owner=$(apk info --who-owns /usr/bin/tunnelctl 2>/dev/null || :)
    case "$owner" in
        "/usr/bin/tunnelctl is owned by tunnelctl-"*) ;;
        *)
            umask 077
            mkdir -p /var/backups
            backup=$(mktemp -d /var/backups/tunnelctl-unmanaged.XXXXXXXX)
            mv /usr/bin/tunnelctl "$backup/tunnelctl"
            echo "WARNING: Unmanaged /usr/bin/tunnelctl moved intact to $backup/tunnelctl before package installation. Review/migrate it explicitly; uninstall preserves this backup." >&2
            ;;
    esac
fi
