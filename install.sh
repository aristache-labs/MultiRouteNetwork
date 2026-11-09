#!/bin/sh
# Installs MultiRouteNetwork on macOS or Linux, the easiest way the machine allows:
#
#   curl -fsSL https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.sh | sh
#
# Linux: the system's package manager (apt, dnf, yum, zypper, apk, pacman) from MultiRouteNetwork's signed
# repositories, so it updates with everything else; otherwise Homebrew; otherwise the release tarball.
# macOS: Homebrew if it is installed, otherwise the release tarball.
# Then it starts the tray. MULTIROUTE_NETWORK_METHOD=tarball skips the package managers, MULTIROUTE_NETWORK_NO_START=1
# leaves the tray off.
#
# Uninstall, however it was installed:
#
#   curl -fsSL https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.sh | sh -s -- uninstall
set -eu

REPO=https://github.com/aristache-labs/MultiRouteNetwork
BASE=${MULTIROUTE_NETWORK_PACKAGE_BASE_URL:-}
FPR=0168C7AB77721656904811AE904F61E7F4F1C2A1

say() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
die() { printf 'MultiRouteNetwork install: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }
fetch() { # fetch URL: the body on stdout
    if have curl; then curl -fsSL "$1"; elif have wget; then wget -qO- "$1"; else die "needs curl or wget"; fi
}

if [ "$(id -u)" = 0 ]; then SUDO=
elif have sudo; then SUDO=sudo
elif have doas; then SUDO=doas
else die "run it as root, or install sudo"; fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# ---- the release tarball, for any CPU MultiRouteNetwork is built for
tarball() {
    if [ "$(uname -s)" = Darwin ]; then
        name=macos-universal
    else
        case $(uname -m) in
            i?86) name=linux-i686 ;;
            armv8l) name=linux-armv7l ;;
            *) name=linux-$(uname -m) ;;
        esac
    fi
    say "Downloading MultiRouteNetwork ($name)"
    fetch "$REPO/releases/latest/download/MultiRouteNetwork-$name.tar.gz" | tar -xz -C "$tmp" MultiRouteNetwork ||
        die "no download for $name; see $REPO/releases"
    if [ "$(uname -s)" = Darwin ]; then
        $SUDO "$tmp/MultiRouteNetwork" install   # copies it to /usr/local/bin, lets the tray start without a password
    else
        $SUDO install -D -m 755 "$tmp/MultiRouteNetwork" /usr/local/bin/MultiRouteNetwork
        if have setcap; then
            $SUDO setcap cap_net_admin,cap_net_raw+ep /usr/local/bin/MultiRouteNetwork
        else
            say "setcap not found: install libcap, then run: sudo setcap cap_net_admin,cap_net_raw+ep /usr/local/bin/MultiRouteNetwork"
        fi
    fi
}

# ---- Homebrew (as the user: brew refuses root)
homebrew() {
    say "Installing with Homebrew"
    brew install "${MULTIROUTE_NETWORK_BREW_TAP:?set MULTIROUTE_NETWORK_BREW_TAP}/MultiRouteNetwork"
    bin=$(brew --prefix)/bin/MultiRouteNetwork
    if [ "$(uname -s)" = Darwin ]; then
        $SUDO "$bin" install
    else
        $SUDO setcap cap_net_admin,cap_net_raw+ep "$(readlink -f "$bin")"
    fi
}

# ---- the system package manager, from the signed repositories
apt_install() {
    say "Installing with apt"
    fetch "$BASE/apt/key.gpg" > "$tmp/key.gpg"
    $SUDO install -D -m 644 "$tmp/key.gpg" /usr/share/keyrings/MultiRouteNetwork.gpg
    echo "deb [signed-by=/usr/share/keyrings/MultiRouteNetwork.gpg] $BASE/apt stable main" | $SUDO tee /etc/apt/sources.list.d/MultiRouteNetwork.list >/dev/null
    $SUDO apt-get update -qq
    $SUDO env DEBIAN_FRONTEND=noninteractive apt-get install -y -qq MultiRouteNetwork
}
rpm_install() { # rpm_install dnf|yum
    say "Installing with $1"
    fetch "$BASE/rpm/MultiRouteNetwork.repo" > "$tmp/MultiRouteNetwork.repo"
    $SUDO install -m 644 "$tmp/MultiRouteNetwork.repo" /etc/yum.repos.d/MultiRouteNetwork.repo
    $SUDO "$1" install -y MultiRouteNetwork
}
zypper_install() {
    say "Installing with zypper"
    $SUDO zypper --non-interactive lr MultiRouteNetwork >/dev/null 2>&1 || $SUDO zypper --non-interactive addrepo "$BASE/rpm/MultiRouteNetwork.repo"
    $SUDO zypper --non-interactive --gpg-auto-import-keys refresh MultiRouteNetwork
    $SUDO zypper --non-interactive install MultiRouteNetwork
}
apk_install() {
    say "Installing with apk"
    fetch "$BASE/apk/MultiRouteNetwork.rsa.pub" > "$tmp/MultiRouteNetwork.rsa.pub"
    $SUDO install -m 644 "$tmp/MultiRouteNetwork.rsa.pub" /etc/apk/keys/MultiRouteNetwork.rsa.pub
    grep -qx "$BASE/apk" /etc/apk/repositories || echo "$BASE/apk" | $SUDO tee -a /etc/apk/repositories >/dev/null
    $SUDO apk add MultiRouteNetwork
}
pacman_install() {
    say "Installing with pacman"
    # a fresh system may have no local pacman key yet, which signing ours needs
    $SUDO pacman-key --list-secret-keys 2>/dev/null | grep -q '^sec' || $SUDO pacman-key --init
    if ! $SUDO pacman-key --list-keys "$FPR" >/dev/null 2>&1; then
        fetch "$BASE/apt/key.asc" > "$tmp/key.asc"
        $SUDO pacman-key --add "$tmp/key.asc"
        $SUDO pacman-key --lsign-key "$FPR"
    fi
    grep -q '^\[MultiRouteNetwork\]' /etc/pacman.conf ||
        printf '\n[MultiRouteNetwork]\nSigLevel = Required\nServer = %s/arch/$arch\n' "$BASE" | $SUDO tee -a /etc/pacman.conf >/dev/null
    $SUDO pacman -Sy --needed --noconfirm MultiRouteNetwork
}

# ---- uninstall: stop it, then remove it the way it was installed, with the repository it came from
uninstall() {
    say "Stopping MultiRouteNetwork"
    if have MultiRouteNetwork; then
        MultiRouteNetwork quit >/dev/null 2>&1 || true           # the tray
        $SUDO MultiRouteNetwork off >/dev/null 2>&1 || true      # system-wide mode started from a terminal
        MultiRouteNetwork autostart off >/dev/null 2>&1 || true  # start at sign-in
        sleep 2
    fi
    if [ "$(uname -s)" = Darwin ]; then
        for exe in /opt/MultiRouteNetwork/bin/MultiRouteNetwork /usr/local/bin/MultiRouteNetwork; do
            [ -x "$exe" ] && { $SUDO "$exe" uninstall || true; break; }   # the root-owned copy and the sudoers rule
        done
        if have brew && brew list --formula MultiRouteNetwork >/dev/null 2>&1; then
            say "Removing with Homebrew"
            brew uninstall MultiRouteNetwork
        fi
        $SUDO rm -rf /var/run/MultiRouteNetwork "/Library/Application Support/MultiRouteNetwork" "$HOME/Library/Logs/MultiRouteNetwork.log"
    else
        if have dpkg && dpkg -s MultiRouteNetwork >/dev/null 2>&1; then
            say "Removing with apt"
            $SUDO env DEBIAN_FRONTEND=noninteractive apt-get remove -y -qq MultiRouteNetwork
            $SUDO rm -f /etc/apt/sources.list.d/MultiRouteNetwork.list /usr/share/keyrings/MultiRouteNetwork.gpg
        elif have rpm && rpm -q MultiRouteNetwork >/dev/null 2>&1; then
            if have zypper; then
                say "Removing with zypper"
                $SUDO zypper --non-interactive remove MultiRouteNetwork
                $SUDO zypper --non-interactive removerepo MultiRouteNetwork >/dev/null 2>&1 || true
            else
                pm=dnf; have dnf || pm=yum
                say "Removing with $pm"
                $SUDO "$pm" remove -y MultiRouteNetwork
                $SUDO rm -f /etc/yum.repos.d/MultiRouteNetwork.repo
            fi
        elif have apk && apk info -e MultiRouteNetwork >/dev/null 2>&1; then
            say "Removing with apk"
            $SUDO apk del MultiRouteNetwork
            $SUDO sed -i "\#^$BASE/apk\$#d" /etc/apk/repositories
            $SUDO rm -f /etc/apk/keys/MultiRouteNetwork.rsa.pub
        elif have pacman && pacman -Q MultiRouteNetwork >/dev/null 2>&1; then
            say "Removing with pacman"
            $SUDO pacman -R --noconfirm MultiRouteNetwork
            $SUDO sed -i '/^\[MultiRouteNetwork\]$/,/^Server = .*MultiRouteNetwork/d' /etc/pacman.conf
            $SUDO pacman-key --delete "$FPR" >/dev/null 2>&1 || true
        fi
        if have brew && brew list --formula MultiRouteNetwork >/dev/null 2>&1; then
            say "Removing with Homebrew"
            brew uninstall MultiRouteNetwork
        fi
        $SUDO rm -f /usr/local/bin/MultiRouteNetwork   # the tarball
        rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/MultiRouteNetwork"
    fi
    hash -r 2>/dev/null || true
    have MultiRouteNetwork && die "a MultiRouteNetwork is still at $(command -v MultiRouteNetwork); remove it by hand"
    say "MultiRouteNetwork is uninstalled. Your network settings are back to normal."
}

case ${1:-} in
    uninstall|--uninstall) uninstall; exit 0 ;;
    '') ;;
    *) die "unknown option $1 (use: uninstall)" ;;
esac

# ---- pick the way
method=${MULTIROUTE_NETWORK_METHOD:-auto}
if [ "$(uname -s)" = Darwin ]; then
    # a running tray holds the old binary: stop it before replacing it
    pgrep -qf 'MultiRouteNetwork tray' && { MultiRouteNetwork quit 2>/dev/null || true; sleep 2; }
    $SUDO mkdir -p /usr/local/bin   # where `MultiRouteNetwork install` puts its root-owned copy
    if [ "$method" = auto ] && [ -n "${MULTIROUTE_NETWORK_BREW_TAP:-}" ] && have brew && [ "$(id -u)" != 0 ]; then homebrew; else tarball; fi
elif [ "$(uname -s)" = Linux ]; then
    if [ -f /etc/NIXOS ]; then
        die "on NixOS use the module instead: $REPO#nixos"
    elif [ "$method" != auto ] || [ -z "$BASE" ]; then tarball
    elif have apt-get && have dpkg && case $(dpkg --print-architecture) in amd64|arm64|armhf|i386|loong64|ppc64el|riscv64|s390x) true ;; *) false ;; esac; then apt_install
    elif have dnf; then rpm_install dnf
    elif have yum; then rpm_install yum
    elif have zypper; then zypper_install
    elif have apk; then apk_install
    elif have pacman; then pacman_install
    elif have brew && [ "$(id -u)" != 0 ]; then homebrew
    else tarball; fi
else
    die "this script is for macOS and Linux; on Windows run in PowerShell: irm https://raw.githubusercontent.com/aristache-labs/MultiRouteNetwork/main/install.ps1 | iex"
fi

have MultiRouteNetwork || PATH=$PATH:/usr/local/bin
say "Installed $(MultiRouteNetwork version)"

# ---- start the tray: on a Mac, or on a Linux desktop session that isn't root's
if [ "${MULTIROUTE_NETWORK_NO_START:-}" = 1 ]; then
    say "Start it with: MultiRouteNetwork tray"
elif [ "$(uname -s)" = Darwin ] || { [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ] && [ "$(id -u)" != 0 ]; }; then
    nohup MultiRouteNetwork tray >/dev/null 2>&1 &
    say "Started: look for the MultiRouteNetwork icon in the menu bar or tray. To start it at sign-in: MultiRouteNetwork autostart on"
else
    say "Start it with: MultiRouteNetwork tray (from your desktop session), or MultiRouteNetwork run (any terminal, as root)"
fi
