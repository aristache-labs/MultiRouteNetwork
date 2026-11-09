#!/bin/sh
# Lets MultiRouteNetwork create its virtual adapter without running as root. File capabilities are
# dropped whenever the binary is replaced, so this runs on every install and upgrade.
if command -v setcap >/dev/null 2>&1; then
    setcap cap_net_admin,cap_net_raw+ep /usr/bin/MultiRouteNetwork || echo "MultiRouteNetwork: setcap failed; run: sudo setcap cap_net_admin,cap_net_raw+ep /usr/bin/MultiRouteNetwork"
else
    echo "MultiRouteNetwork: setcap not found; install libcap and run: sudo setcap cap_net_admin,cap_net_raw+ep /usr/bin/MultiRouteNetwork"
fi

# Package repository URLs are configured by the operator. Never rewrite them
# to a domain inferred from the project name.
exit 0
