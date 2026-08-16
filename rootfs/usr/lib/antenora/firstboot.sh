#!/usr/bin/env bash
# Antenora first boot: ZRAM, SSH key, firewall.
# Invoked once by the s6 boot bundle (oneshot) on initial boot.
set -euo pipefail

STAMP="/var/lib/antenora/firstboot.done"
[ -f "$STAMP" ] && exit 0

# --- ZRAM (lz4) — half of physical RAM, no swap partition ---------------
if command -v zramctl >/dev/null 2>&1; then
    RAM_KB=$(awk '/MemTotal/{print $2}' /proc/meminfo)
    HALF=$((RAM_KB / 2))
    zramctl --find --size "$((HALF))K" --algorithm lz4 >/dev/null 2>&1 || true
    if [ -e /dev/zram0 ]; then
        mkswap /dev/zram0 >/dev/null 2>&1 || true
        swapon /dev/zram0 >/dev/null 2>&1 || true
    fi
fi

# --- SSH key generation for every human user -----------------------------
for home in /home/*; do
    user=$(basename "$home")
    id "$user" >/dev/null 2>&1 || continue
    keyfile="$home/.ssh/id_ed25519"
    if [ ! -f "$keyfile" ]; then
        mkdir -p "$home/.ssh"
        ssh-keygen -t ed25519 -f "$keyfile" -N '' -C "$user@$(hostname)" -q
        chown -R "$user":"$user" "$home/.ssh"
    fi
done

# --- Firewall ------------------------------------------------------------
if command -v nft >/dev/null 2>&1 && [ -f /etc/nftables.conf ]; then
    nft -f /etc/nftables.conf
fi

mkdir -p "$(dirname "$STAMP")"
touch "$STAMP"
