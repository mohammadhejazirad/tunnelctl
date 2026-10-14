#!/usr/bin/env bash
# Called only by the tag-only, gated release job. Never use --clobber.
set -euo pipefail
assets=${1:?Usage: publish.sh ASSETS TAG}
tag=${2:?}
test "${GITHUB_EVENT_NAME:-}" = push
test "${GITHUB_REF:-}" = "refs/tags/$tag"
go run ./cmd/packaging --mode verify-assets --tag "$tag" --out "$assets"
# Fail on API/authentication errors, not just on existing releases.
existing=$(gh release list --limit 1000 --json tagName)
if jq -e --arg tag "$tag" 'any(.[]; .tagName == $tag)' <<< "$existing" >/dev/null; then
  echo 'An existing draft or published release is never overwritten. Review the documented failed-draft recovery procedure.' >&2
  exit 1
fi
files=()
while read -r digest name; do
  test "${#digest}" = 64
  test "$name" = "${name##*/}"
  files+=("$assets/$name")
done < "$assets/checksums.txt"
test "${#files[@]}" = 23
files+=("$assets/checksums.txt")
flags=(--latest=false)
prerelease=false
if [[ "$tag" == *-* ]]; then flags+=(--prerelease); prerelease=true; fi
# Failed upload/verification leaves only a draft, never a public partial matrix.
gh release create "$tag" "${files[@]}" --verify-tag --draft "${flags[@]}" --title "$tag" --notes-file "$assets/release-notes.md"
verify_dir=$(mktemp -d)
trap 'rm -rf "$verify_dir"' EXIT
gh release download "$tag" --dir "$verify_dir"
cmp "$assets/checksums.txt" "$verify_dir/checksums.txt"
go run ./cmd/packaging --mode verify-assets --tag "$tag" --out "$verify_dir"
latest=true
if [[ "$prerelease" == true ]]; then latest=false; fi
gh release edit "$tag" --draft=false --prerelease="$prerelease" --latest="$latest"
echo "Complete release published: $tag"
