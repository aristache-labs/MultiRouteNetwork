#!/bin/sh
# Signs a MultiRoute.app from the inside out, builds a DMG, notarizes it and staples the ticket.
# Secrets stay in the environment. The script prints no certificate or key material.
# Required: APPLE_CERTIFICATE_BASE64, APPLE_CERTIFICATE_PASSWORD,
# APPLE_NOTARY_ISSUER_ID, APPLE_NOTARY_KEY_ID, APPLE_NOTARY_KEY_BASE64.
set -eu
app=${1:?app bundle}
dmg=${2:?dmg path}
for name in APPLE_CERTIFICATE_BASE64 APPLE_CERTIFICATE_PASSWORD APPLE_NOTARY_ISSUER_ID APPLE_NOTARY_KEY_ID APPLE_NOTARY_KEY_BASE64; do
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "missing signing secret: $name" >&2
    exit 1
  fi
done
keychain="$RUNNER_TEMP/multiroute.keychain"
cert="$RUNNER_TEMP/developer-id.p12"
notary="$RUNNER_TEMP/notary-key.p8"
printf '%s' "$APPLE_CERTIFICATE_BASE64" | base64 --decode > "$cert"
printf '%s' "$APPLE_NOTARY_KEY_BASE64" | base64 --decode > "$notary"
security create-keychain -p multiroute "$keychain"
security unlock-keychain -p multiroute "$keychain"
security import "$cert" -k "$keychain" -P "$APPLE_CERTIFICATE_PASSWORD" -T /usr/bin/codesign
security set-key-partition-list -S apple-tool:,apple: -s -k multiroute "$keychain" >/dev/null
identity=$(security find-identity -v -p codesigning "$keychain" | awk -F\" 'NR==1 {print $2}')
test -n "$identity"
sign() { codesign --force --options runtime --timestamp --sign "$identity" --keychain "$keychain" "$1"; }
sign "$app/Contents/MacOS/MultiRouteNetwork"
sign "$app/Contents/MacOS/MultiRoute"
sign "$app"
stage=$(mktemp -d)
cp -R "$app" "$stage/MultiRoute.app"
ln -s /Applications "$stage/Applications"
hdiutil create -volname MultiRoute -srcfolder "$stage" -ov -format UDZO "$dmg"
sign "$dmg"
xcrun notarytool submit "$dmg" --key "$notary" --key-id "$APPLE_NOTARY_KEY_ID" --issuer "$APPLE_NOTARY_ISSUER_ID" --wait
xcrun stapler staple "$dmg"
rm -f "$cert" "$notary"
echo "$dmg"
