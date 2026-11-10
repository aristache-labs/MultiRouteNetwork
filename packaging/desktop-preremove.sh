#!/bin/sh
# Removes the service unit and leaves the routing history on disk.
set -eu
if command -v systemctl >/dev/null 2>&1; then
    systemctl disable --now multiroute.service || true
fi
exit 0
