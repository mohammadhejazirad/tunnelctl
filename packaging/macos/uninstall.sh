#!/bin/sh
set -eu
PATH=/usr/bin:/bin:/usr/sbin:/sbin
export PATH
# Invoked explicitly by the administrator, never by installation or the app.
receipt=io.github.mohammadhejazirad.tunnelctl
pkgutil --pkg-info "$receipt" >/dev/null
target=/usr/local/bin/tunnelctl
if [ -L "$target" ] && [ "$(readlink "$target")" = '/Library/Application Support/tunnelctl/tunnelctl' ]; then
    unlink "$target"
elif [ -e "$target" ] || [ -L "$target" ]; then
    echo 'Command path is no longer owned by tunnelctl; leaving it unchanged.' >&2
fi
# Remove only files in this receipt, not arbitrary contents of the payload tree.
pkgutil --only-files --files "$receipt" | while IFS= read -r relative; do
    case "$relative" in
        "Library/Application Support/tunnelctl/"*)
            if [ -f "/$relative" ] || [ -L "/$relative" ]; then rm -f "/$relative"; fi ;;
        *) echo 'Unexpected receipt entry; refusing removal.' >&2; exit 1 ;;
    esac
done
pkgutil --forget "$receipt"
echo 'Application removed. User profiles, settings, caches and external cores were preserved.'
