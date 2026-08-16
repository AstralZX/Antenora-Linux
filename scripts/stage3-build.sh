#!/usr/bin/env bash
# =============================================================================
# stage3-build.sh - assemble the Antenora Stage3 tarball
# =============================================================================
# Produces a self-hosting base system tarball containing s6 + s6-rc, the
# Antenora overlay (configs, motd, firewall, fastfetch), the initramfs tooling
# and the Dante package manager. This solves the bootstrap chicken-and-egg
# problem: Dante can install itself because it ships inside Stage3.
# =============================================================================
set -euo pipefail

VERSION="${1:-1.0.0}"
ARCH="${2:-x86_64}"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUTDIR="$REPO_DIR/dist"
ROOTFS="$OUTDIR/stage3-root"
TARBALL="$OUTDIR/stage3-${VERSION}-${ARCH}.tar.xz"

echo ":: Building Antenora Stage3 $VERSION ($ARCH)"

rm -rf "$ROOTFS"
mkdir -p "$ROOTFS"/{bin,sbin,usr/bin,usr/sbin,etc,dev,proc,sys,run,tmp,var/lib/dante,var/cache/dante}

# --- s6 service sources ---------------------------------------------------
mkdir -p "$ROOTFS/etc/s6-rc/source"
[ -d "$REPO_DIR/s6-services" ] && cp -r "$REPO_DIR/s6-services"/* "$ROOTFS/etc/s6-rc/source/"

# --- Antenora overlay (configs, motd, firewall, fastfetch, zshrc, ...) -----
cp -a "$REPO_DIR/rootfs/." "$ROOTFS/"
mkdir -p "$ROOTFS/etc/sysctl.d"
cp -a "$REPO_DIR/sysctl/." "$ROOTFS/etc/sysctl.d/"

# --- initramfs + boot tooling --------------------------------------------
mkdir -p "$ROOTFS/usr/lib/antenora"
install -Dm755 "$REPO_DIR/scripts/mkinitramfs.sh" "$ROOTFS/usr/lib/antenora/mkinitramfs.sh"
install -Dm755 "$REPO_DIR/scripts/s6-init-setup.sh" "$ROOTFS/usr/lib/antenora/s6-init-setup.sh"
install -Dm755 "$REPO_DIR/scripts/install.sh" "$ROOTFS/usr/local/bin/install-antenora"
cp -a "$REPO_DIR/boot" "$ROOTFS/usr/lib/antenora/boot"

# --- identity --------------------------------------------------------------
cat > "$ROOTFS/etc/antenora-release" <<EOF
Antenora Linux $VERSION "Inferno" — The Ninth Circle of Speed
EOF

cat > "$ROOTFS/etc/os-release" <<EOF
NAME="Antenora Linux"
VERSION="$VERSION (Inferno)"
ID=antenora
PRETTY_NAME="Antenora Linux $VERSION"
ANSI_COLOR="1;31"
HOME_URL="https://github.com/AstralZX/Antenora-Linux"
EOF

cat > "$ROOTFS/etc/dante/dante.conf" <<EOF
# Dante Package Manager Configuration
BINARY="YES"
MAKEFLAGS="-j\$(nproc)"
REPO_URL="https://github.com/AstralZX/antenora-packages.git"
DUR_URL="https://github.com/AstralZX/antenora-dur.git"
BINARY_MIRROR="https://cdn.antenora.org/packages"
MIRROR_FILE="/etc/dante/mirrors.conf"
CLEAN_SOURCE="YES"
KEEP_DEPS="NO"
EOF

# --- build & install the Dante binary into the Stage3 root ---------------
echo ":: Building Dante..."
CGO_ENABLED=0 go -C "$REPO_DIR" build -trimpath -ldflags '-s -w' -o "$ROOTFS/usr/bin/dante" ./cmd/dante

# --- tarball --------------------------------------------------------------
echo ":: Compressing Stage3..."
mkdir -p "$OUTDIR"
tar -cJpf "$TARBALL" -C "$ROOTFS" .

echo ":: Stage3 written: $TARBALL"
ls -lh "$TARBALL"
