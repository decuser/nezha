#!/bin/sh
set -eu

VERSION=13.3.0
SHA256=b6aff9a0f6cda07894b0f8265b81d30949f863327055987b797cfa2953b4f1ea
URL="https://github.com/RaspberryPiFoundation/blockly/releases/download/blockly-v${VERSION}/blockly-${VERSION}.tgz"

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
dest="$here/web/vendor/blockly"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
archive="$tmp/blockly.tgz"

if command -v curl >/dev/null 2>&1; then
    curl -L --fail --show-error -o "$archive" "$URL"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$archive" "$URL"
else
    echo "need curl or wget" >&2
    exit 1
fi

echo "$SHA256  $archive" | sha256sum -c -
mkdir -p "$tmp/unpack"
tar -xzf "$archive" -C "$tmp/unpack"

root=$(find "$tmp/unpack" -type f -name blockly_compressed.js -print | head -1 | sed 's#/blockly_compressed.js$##')
if [ -z "$root" ]; then
    echo "blockly_compressed.js not found in release archive" >&2
    exit 1
fi

rm -rf "$dest"
mkdir -p "$dest/msg"
cp "$root/blockly_compressed.js" "$dest/"
cp "$root/msg/en.js" "$dest/msg/"
cp -R "$root/media" "$dest/media"
cp "$here/third_party/Blockly-LICENSE" "$dest/LICENSE"

echo "vendored Blockly $VERSION in $dest"
