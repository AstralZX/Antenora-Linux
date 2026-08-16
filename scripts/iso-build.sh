#!/usr/bin/env bash
# =============================================================================
# iso-build.sh — build a bootable Antenora ISO from source.
# =============================================================================
# No Arch. No mkarchiso. No pacman. No systemd.
# Everything is compiled from source by Dante, booted by s6-linux-init, and
# wrapped into a hybrid BIOS/UEFI ISO by grub-mkrescue (GNU) + xorriso.
#
# Pipeline:
#   0. verify an independent host toolchain exists
#   1. bootstrap the Antenora toolchain into a sysroot (scripts/toolchain.sh)
#   2. build base-system + kernel + firmware from source (Dante)
#   3. apply the Antenora overlay (rootfs/)
#   4. generate a busybox initramfs (scripts/mkinitramfs.sh)
#   5. squashfs the sysroot, assemble the ISO
# =============================================================================
set -euo pipefail

VERSION="${1:-1.0.0}"
ARCH="${2:-x86_64}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYSROOT="${ANTENORA_SYSROOT:-$ROOT/dist/sysroot}"
WORK="$ROOT/dist/iso-work"
OUT="$ROOT/dist/iso"
ISO="$OUT/antenora-$VERSION-$ARCH.iso"
DANTE="$ROOT/dist/dante"

red()  { printf '\033[1;31m:: %s\033[0m\n' "$*"; }
die()  { red "FATAL: $*"; exit 1; }

# --- 0. host toolchain ------------------------------------------------------
red "Verifying host toolchain..."
for t in gcc make tar xz cpio gzip git go grub-mkrescue xorriso mksquashfs; do
    command -v "$t" >/dev/null 2>&1 || die "missing host tool: $t"
done

# --- build dante -----------------------------------------------------------
if [ ! -x "$DANTE" ]; then
    red "Building Dante..."
    CGO_ENABLED=0 go -C "$ROOT" build -trimpath -ldflags '-s -w' -o "$DANTE" ./cmd/dante
fi

# --- 1. toolchain bootstrap ------------------------------------------------
if [ ! -x "$SYSROOT/usr/bin/gcc" ]; then
    red "Bootstrapping the Antenora toolchain (from source)..."
    "$ROOT/scripts/toolchain.sh"
fi

# --- 2. base system from source --------------------------------------------
red "Building the Antenora base system from source..."
export DANTE_ROOT="$SYSROOT"
"$DANTE" install base-system
"$DANTE" install linux-cachyos-bore
"$DANTE" install linux-firmware
"$DANTE" install busybox squashfs-tools

# --- 3. Antenora overlay ---------------------------------------------------
red "Applying the Antenora overlay..."
cp -a "$ROOT/rootfs/." "$SYSROOT/"
mkdir -p "$SYSROOT/etc/sysctl.d"
cp -a "$ROOT/sysctl/." "$SYSROOT/etc/sysctl.d/"
install -Dm755 "$DANTE" "$SYSROOT/usr/bin/dante"
install -Dm755 "$ROOT/scripts/install.sh" "$SYSROOT/usr/local/bin/install-antenora"
# ship the initramfs tooling so the installed system can rebuild its initramfs
install -Dm755 "$ROOT/scripts/mkinitramfs.sh" "$SYSROOT/usr/lib/antenora/mkinitramfs.sh"
cp -a "$ROOT/boot" "$SYSROOT/usr/lib/antenora/boot"

# --- 3b. s6-linux-init (PID 1) setup ---------------------------------------
red "Installing s6-linux-init as PID 1..."
install -Dm755 "$ROOT/scripts/s6-init-setup.sh" "$SYSROOT/usr/lib/antenora/s6-init-setup.sh"
chroot "$SYSROOT" /bin/sh /usr/lib/antenora/s6-init-setup.sh

# --- 4. initramfs ----------------------------------------------------------
red "Building the busybox initramfs..."
"$ROOT/scripts/mkinitramfs.sh" "$SYSROOT" "$WORK/initramfs.img"

# --- 5. squashfs + ISO assembly --------------------------------------------
red "Assembling the ISO..."
rm -rf "$WORK/iso"
mkdir -p "$WORK/iso/antenora" "$WORK/iso/boot/grub"
mksquashfs "$SYSROOT" "$WORK/iso/antenora/rootfs.squashfs" -comp xz -noappend -no-progress

KERNEL="$SYSROOT/boot/vmlinuz-linux-cachyos-bore"
[ -f "$KERNEL" ] || KERNEL="$(ls "$SYSROOT"/boot/vmlinuz* 2>/dev/null | head -1)"
[ -n "$KERNEL" ] || die "kernel image not found in sysroot"
cp "$KERNEL" "$WORK/iso/antenora/vmlinuz"
cp "$WORK/initramfs.img" "$WORK/iso/antenora/initramfs.img"

cp "$ROOT/boot/grub/grub.cfg" "$WORK/iso/boot/grub/grub.cfg"
cp -a "$ROOT/theme/grub-theme" "$WORK/iso/boot/grub/theme"

mkdir -p "$OUT"
grub-mkrescue -o "$ISO" "$WORK/iso"

red "ISO written: $ISO"
ls -lh "$ISO"
