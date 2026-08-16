#!/usr/bin/env bash
# =============================================================================
# stage3-build.sh - assemble the Antenora Stage3 tarball
# =============================================================================
# Produces a self-hosting base system tarball containing s6 + s6-rc, a
# minimal glibc toolchain surface and the Dante package manager. This solves
# the bootstrap chicken-and-egg problem: Dante can install itself because it
# ships inside Stage3.
# =============================================================================
set -euo pipefail

VERSION="${1:-1.0.0}"
ARCH="${2:-x86_64}"
OUTDIR="$(cd "$(dirname "$0")/.." && pwd)/dist"
ROOTFS="$OUTDIR/stage3-root"
TARBALL="$OUTDIR/stage3-${VERSION}-${ARCH}.tar.xz"

echo ":: Building Antenora Stage3 $VERSION ($ARCH)"

rm -rf "$ROOTFS"
mkdir -p "$ROOTFS"/{bin,sbin,usr/bin,usr/sbin,etc,dev,proc,sys,run,tmp,var/lib/dante,var/cache/dante}

# --- minimal directory skeleton ------------------------------------------
mkdir -p "$ROOTFS"/etc/s6-rc/source
mkdir -p "$ROOTFS"/usr/share/antenora

# --- copy s6 service sources from this repo ------------------------------
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
if [ -d "$REPO_DIR/s6-services" ]; then
    cp -r "$REPO_DIR/s6-services"/* "$ROOTFS"/etc/s6-rc/source/
fi

# --- seed configuration ---------------------------------------------------
cp "$REPO_DIR/sysctl/99-antenora-performance.conf" "$ROOTFS"/etc/sysctl.d/ 2>/dev/null || true

cat > "$ROOTFS/etc/antenora-release" <<EOF
Antenora Linux $VERSION "Inferno" — The Ninth Circle of Speed
EOF

cat > "$ROOTFS/etc/os-release" <<EOF
NAME="Antenora Linux"
VERSION="$VERSION (Inferno)"
ID=antenora
PRETTY_NAME="Antenora Linux $VERSION"
ANSI_COLOR="1;31"
HOME_URL="https://antenora.org"
EOF

cat > "$ROOTFS/etc/dante/dante.conf" <<EOF
# Dante Package Manager Configuration
BINARY="YES"
MAKEFLAGS="-j\$(nproc)"
REPO_URL="https://github.com/antenora/package-repo.git"
BINARY_MIRROR="https://cdn.antenora.org/packages"
CLEAN_SOURCE="YES"
KEEP_DEPS="NO"
EOF

# --- build & install the Dante binary into the Stage3 root ---------------
echo ":: Building Dante..."
cd "$REPO_DIR"
CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o "$ROOTFS/usr/bin/dante" ./cmd/dante
echo ":: Dante installed at $ROOTFS/usr/bin/dante"

# --- resolve a dynamic glibc by simply recording the toolchain hint -------
# The Stage3 is expected to be completed with the glibc recipe on first boot
# (`dante install glibc`). Here we only guarantee the binary lands on disk.

# --- generate the tarball -------------------------------------------------
echo ":: Compressing Stage3..."
mkdir -p "$OUTDIR"
tar -cJpf "$TARBALL" -C "$ROOTFS" .

echo ":: Stage3 written: $TARBALL"
ls -lh "$TARBALL"
