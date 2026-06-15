#!/usr/bin/env bash
#
# Render the Homebrew formula from scripts/claude-openai-proxy.rb.tmpl by
# filling in the version and the per-platform sha256 sums.
#
# Usage:
#   scripts/update-formula.sh <version> <checksums-file> <output-formula>
#
#   <version>        release version without the leading "v" (e.g. 1.2.3)
#   <checksums-file> sha256sum-format file listing the release archives
#   <output-formula> path to write the rendered Formula/*.rb to
set -euo pipefail

VERSION="${1:?version required (without leading v)}"
CHECKSUMS="${2:?checksums.txt path required}"
OUTPUT="${3:?output formula path required}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="${SCRIPT_DIR}/claude-openai-proxy.rb.tmpl"

# Strip a stray leading "v" if the caller passed the raw tag.
VERSION="${VERSION#v}"

sha_for() {
  local name="$1" sum
  sum="$(awk -v f="$name" '$2 == f {print $1}' "$CHECKSUMS")"
  if [ -z "$sum" ]; then
    echo "error: no checksum for '$name' in $CHECKSUMS" >&2
    exit 1
  fi
  printf '%s' "$sum"
}

DARWIN_ARM64="$(sha_for "claude-openai-proxy_${VERSION}_darwin_arm64.tar.gz")"
DARWIN_AMD64="$(sha_for "claude-openai-proxy_${VERSION}_darwin_amd64.tar.gz")"
LINUX_ARM64="$(sha_for "claude-openai-proxy_${VERSION}_linux_arm64.tar.gz")"
LINUX_AMD64="$(sha_for "claude-openai-proxy_${VERSION}_linux_amd64.tar.gz")"

mkdir -p "$(dirname "$OUTPUT")"
sed \
  -e "s|__VERSION__|${VERSION}|g" \
  -e "s|__SHA256_DARWIN_ARM64__|${DARWIN_ARM64}|g" \
  -e "s|__SHA256_DARWIN_AMD64__|${DARWIN_AMD64}|g" \
  -e "s|__SHA256_LINUX_ARM64__|${LINUX_ARM64}|g" \
  -e "s|__SHA256_LINUX_AMD64__|${LINUX_AMD64}|g" \
  "$TEMPLATE" > "$OUTPUT"

echo "Wrote $OUTPUT (version $VERSION)"
