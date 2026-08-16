#!/usr/bin/env bash
# =============================================================================
# mkinitramfs.sh — build a busybox-based initramfs. No systemd, no dracut.
#
# Usage: mkinitramfs.sh <sysroot> <output.img> [init-script]
#   init-script defaults to boot/live-boot/init (live media); use
#   boot/disk-boot/init for an installed system.
# =============================================================================
set -euo pipefail

SYSROOT="${1:?usage: mkinitramfs.sh <sysroot> <output>}"
OUT="${2:?usage: mkinitramfs.sh <sysroot> <output>}"
INIT="${3:-boot/live-boot/init}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

mkdir -p "$STAGE"/{bin,proc,sys,dev,run,newroot,etc,lib}

# --- static busybox + applet symlinks --------------------------------------
BUSYBOX="$SYSROOT/bin/busybox"
[ -f "$BUSYBOX" ] || { echo "error: $BUSYBOX not found (build busybox first)"; exit 1; }
cp "$BUSYBOX" "$STAGE/bin/busybox"
chmod +x "$STAGE/bin/busybox"
for a in sh mount umount switch_root modprobe mdev sleep cat echo mkdir switch_root; do
    ln -sf busybox "$STAGE/bin/$a"
done

# --- live init -------------------------------------------------------------
install -Dm755 "$ROOT/$INIT" "$STAGE/init"

# --- kernel modules (so the init can load storage/filesystem drivers) ------
if [ -d "$SYSROOT/lib/modules" ]; then
    cp -a "$SYSROOT/lib/modules" "$STAGE/lib/"
fi

# --- pack ------------------------------------------------------------------
mkdir -p "$(dirname "$OUT")"
( cd "$STAGE" && find . -print0 | cpio --null -o -H newc 2>/dev/null | gzip -9 ) > "$OUT"
echo "initramfs: $OUT ($(du -h "$OUT" | cut -f1))"
