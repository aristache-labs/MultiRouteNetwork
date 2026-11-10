#!/bin/sh
# Builds MultiRoute.app for one architecture. Does not start the GUI or sign the bundle.
# Usage: scripts/package-desktop-macos.sh VERSION arm64|amd64 OUT_DIR
set -eu
cd "$(dirname "$0")/.."
version=${1:?version}
arch=${2:?arm64 or amd64}
out=${3:?output directory}
case "$arch" in
  arm64|amd64) ;;
  *) echo "unsupported architecture" >&2; exit 2 ;;
esac
test -f desktop/frontend/dist/index.html
if [ "$(uname -m)" = arm64 ] && [ "$arch" = amd64 ]; then
  export CC="clang -arch x86_64"
fi
stage="$out/MultiRoute.app"
rm -rf "$stage"
mkdir -p "$stage/Contents/MacOS" "$stage/Contents/Resources" "$stage/Contents/Library/LaunchDaemons"
CGO_ENABLED=1 GOARCH="$arch" go build -trimpath -buildvcs=true -ldflags "-X main.version=$version" -o "$stage/Contents/MacOS/MultiRouteNetwork" ./cmd/MultiRouteNetwork
CGO_ENABLED=1 GOARCH="$arch" go build -trimpath -buildvcs=true -ldflags "-X main.version=$version" -o "$stage/Contents/MacOS/MultiRoute" ./desktop
sed "s/VERSION/$version/g" packaging/macos/Info.plist > "$stage/Contents/Info.plist"
cp packaging/macos/dev.aristache.multiroute.daemon.plist "$stage/Contents/Library/LaunchDaemons/dev.aristache.multiroute.daemon.plist"
echo "$stage"
