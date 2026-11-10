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
pem="$RUNNER_TEMP/developer.pem"
notary="$RUNNER_TEMP/notary-key.p8"
printf '%s' "$APPLE_CERTIFICATE_BASE64" | tr -d '[:space:]' | base64 --decode > "$cert"
printf '%s' "$APPLE_NOTARY_KEY_BASE64" | tr -d '[:space:]' | base64 --decode > "$notary"
# OpenSSL 3 seals the PKCS#12 with a SHA-256 MAC. macOS security import
# rejects that MAC and reports a wrong password. Rewrite it as SHA-1.
p12pass=$(printf '%s' "$APPLE_CERTIFICATE_PASSWORD" | tr -d '\r\n')
export MR_P12_PASS="$p12pass"
openssl_bin=
for candidate in /opt/homebrew/opt/openssl@3/bin/openssl /usr/local/opt/openssl@3/bin/openssl; do
  if [ -x "$candidate" ]; then
    openssl_bin=$candidate
    break
  fi
done
if [ -z "$openssl_bin" ]; then
  echo "OpenSSL 3 is required to import the certificate" >&2
  exit 1
fi
"$openssl_bin" pkcs12 -in "$cert" -passin env:MR_P12_PASS -nodes -out "$pem"
"$openssl_bin" pkcs12 -export -in "$pem" -out "$cert" -passout env:MR_P12_PASS -macalg SHA1 || status=$?
rm -f "$pem"
unset MR_P12_PASS
test "${status:-0}" -eq 0
security create-keychain -p multiroute "$keychain"
security set-keychain-settings -lut 21600 "$keychain"
security unlock-keychain -p multiroute "$keychain"
# codesign ignores a keychain that is not on the user search list.
security list-keychains -d user -s "$keychain"
security default-keychain -s "$keychain"
security import "$cert" -k "$keychain" -P "$p12pass" -f pkcs12 -A -T /usr/bin/codesign -T /usr/bin/security
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k multiroute "$keychain" >/dev/null
security unlock-keychain -p multiroute "$keychain"
identity_line=$(security find-identity -v -p codesigning "$keychain" | sed -n 's/^[[:space:]]*[0-9][0-9]*) //p' | head -n 1)
hash=$(printf '%s\n' "$identity_line" | awk '{print $1}')
identity=$(printf '%s\n' "$identity_line" | awk -F'"' '{print $2}')
if ! printf '%s' "$hash" | grep -Eq '^[0-9A-F]{40}$'; then
  echo "no codesigning identity in the temporary keychain" >&2
  security find-identity -v -p codesigning "$keychain" >&2 || true
  exit 1
fi
echo "signing identity $identity"
sign() { codesign --force --keychain "$keychain" --options runtime --timestamp --sign "$hash" "$1"; }
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
