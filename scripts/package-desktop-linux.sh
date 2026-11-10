#!/bin/sh
# Builds the experimental Ubuntu amd64 desktop package. Does not start the GUI.
# Usage: scripts/package-desktop-linux.sh VERSION OUT_DIR
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
version=${1:?version}
out=$(CDPATH= cd -- "$(dirname "$2")" && pwd)/$(basename "$2")
mkdir -p "$out"
stage="$root/build/linux-desktop"
rm -rf "$stage"
mkdir -p "$stage/packaging/icons"
test -f "$root/desktop/frontend/dist/index.html"
(
  cd "$root"
  CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=$version" -o "$stage/MultiRouteNetwork" ./cmd/MultiRouteNetwork
  CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=$version" -o "$stage/MultiRoute" ./desktop
)
cp "$root/packaging/multiroute.service" "$root/packaging/dev.aristache.multiroute.desktop" \
  "$root/packaging/desktop-postinstall.sh" "$root/packaging/desktop-preremove.sh" "$root/packaging/desktop-postremove.sh" \
  "$stage/packaging/"
cp -a "$root/packaging/icons/hicolor" "$stage/packaging/icons/"
cp "$root/LICENSE" "$root/THIRD-PARTY-NOTICES.md" "$root/packaging/nfpm-desktop.yaml" "$stage/"
nfpm=${NFPM:-nfpm}
(
  cd "$stage"
  VERSION="$version" "$nfpm" package -p deb -f nfpm-desktop.yaml -t "$out/"
)
deb=$(find "$out" -name 'MultiRoute_*.deb' -print -quit)
test -n "$deb"
final="$out/MultiRoute-$version-linux-amd64.deb"
mv "$deb" "$final"
echo "$final"
