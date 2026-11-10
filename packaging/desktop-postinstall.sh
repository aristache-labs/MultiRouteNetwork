#!/bin/sh
# Installs the privileged service without turning routing on.
# The desktop user's history in /var/lib/multiroute is left in place.
set -eu
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi
uid="${SUDO_UID:-}"
if [ -n "$uid" ] && [ "$uid" != "0" ]; then
    /usr/bin/MultiRouteNetwork service-install --uid "$uid" --gui /usr/bin/MultiRoute || true
fi
if command -v systemctl >/dev/null 2>&1; then
    systemctl enable multiroute.service || true
    systemctl start multiroute.service || true
fi
exit 0
