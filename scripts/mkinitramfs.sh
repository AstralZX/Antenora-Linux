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

mkdir -p "$STAGE"/{bin,proc,sys,dev,run,newroot,etc,lib,lib64}

# --- busybox + applet symlinks ----------------------------------------------
BUSYBOX="$SYSROOT/bin/busybox"
[ -f "$BUSYBOX" ] || { echo "error: $BUSYBOX not found (build busybox first)"; exit 1; }
cp "$BUSYBOX" "$STAGE/bin/busybox"
chmod +x "$STAGE/bin/busybox"
for a in sh mount umount switch_root modprobe mdev sleep cat echo mkdir; do
    ln -sf busybox "$STAGE/bin/$a"
done

# --- shared libraries (busybox is dynamically linked) -----------------------
if file "$BUSYBOX" | grep -q dynamically; then
    LD_LINUX="$SYSROOT/lib64/ld-linux-x86-64.so.2"
    if [ -f "$LD_LINUX" ]; then
        cp "$LD_LINUX" "$STAGE/lib64/"
        chmod +x "$STAGE/lib64/ld-linux-x86-64.so.2"
    fi
    for lib in $(ldd "$BUSYBOX" 2>/dev/null | awk '/=>/ {print $3}' | grep -v 'not found'); do
        dir="/$(dirname "$lib")"
        mkdir -p "$STAGE$dir"
        cp "$lib" "$STAGE$lib"
        # copy the soname symlink if it differs
        soname=$(readelf -d "$lib" 2>/dev/null | grep 'NEEDED' | head -1 | awk '{print $NF}' | tr -d '[]')
        if [ -n "$soname" ] && [ ! -e "$STAGE$dir/$soname" ]; then
            ln -sf "$(basename "$lib")" "$STAGE$dir/$soname"
        fi
    done
fi

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
