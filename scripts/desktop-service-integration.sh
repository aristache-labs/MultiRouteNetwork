#!/bin/sh
# Starts the privileged service, checks the capture device, exchanges local TCP and UDP, then stops it.
# No external downloads. Requires root. Does not open the GUI.
# Usage: sudo scripts/desktop-service-integration.sh ./MultiRouteNetwork
set -eu
bin=$(CDPATH= cd -- "$(dirname "$1")" && pwd)/$(basename "$1")
if [ "$(id -u)" != "0" ]; then
  echo "desktop service integration needs root" >&2
  exit 1
fi
dir=$(mktemp -d)
pid=""
cleanup() {
  if [ -n "$pid" ]; then
    "$bin" service-call -state-dir "$dir" -op stop >/dev/null 2>&1 || true
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$dir"
}
trap cleanup EXIT
printf 'uid=0\nexe=%s\n' "$bin" > "$dir/authorized-user"
"$bin" service -state-dir "$dir" >/tmp/multiroute-service.log 2>&1 &
pid=$!
ready=0
i=0
while [ "$i" -lt 50 ]; do
  if "$bin" service-call -state-dir "$dir" -op status 2>/dev/null | grep -q 'running=false'; then
    ready=1
    break
  fi
  i=$((i + 1))
  sleep 0.2
done
test "$ready" = 1
"$bin" service-call -state-dir "$dir" -op start
os=$(uname -s)
if [ "$os" = "Linux" ]; then
  ip link show MultiRouteNetwork >/dev/null
elif [ "$os" = "Darwin" ]; then
  netstat -rn -f inet | grep -q '0/1'
else
  echo "unsupported operating system" >&2
  exit 2
fi
python3 - <<'PY'
import socket
tcp = socket.socket(); tcp.bind(("127.0.0.1", 0)); tcp.listen(1)
port = tcp.getsockname()[1]
client = socket.create_connection(("127.0.0.1", port))
conn, _ = tcp.accept(); conn.sendall(b"ok"); conn.close(); client.close(); tcp.close()
udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); udp.bind(("127.0.0.1", 0))
peer = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
peer.sendto(b"ok", udp.getsockname()); udp.recvfrom(16); peer.close(); udp.close()
PY
"$bin" service-call -state-dir "$dir" -op stop
if [ "$os" = "Linux" ]; then
  if ip link show MultiRouteNetwork >/dev/null 2>&1; then
    echo "capture device remained after stop" >&2
    exit 1
  fi
elif [ "$os" = "Darwin" ]; then
  if netstat -rn -f inet | grep -q '0/1'; then
    echo "capture route remained after stop" >&2
    exit 1
  fi
fi
echo "desktop service integration passed"
