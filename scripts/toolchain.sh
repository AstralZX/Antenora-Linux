#!/usr/bin/env bash
# =============================================================================
# toolchain.sh - bootstrap the Antenora build toolchain from nothing.
# =============================================================================
# Builds the toolchain in dependency order (headers -> gmp/mpfr/mpc/isl ->
# binutils -> gcc -> make/autotools/pkgconf/cmake/meson), staging into a
# sysroot and, optionally, installing into the live system. This is the
# chicken-and-egg solver: it gives Dante a host compiler to build the world.
# =============================================================================
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SYSROOT="${DANTE_SYSROOT:-$REPO_DIR/dist/sysroot}"
JOBS="$(nproc)"

ORDER=(
    linux-api-headers
    zlib
    gmp
    mpfr
    mpc
    isl
    binutils
    gcc
    make
    m4
    autoconf
    automake
    libtool
    pkgconf
    flex
    bison
    patch
    ninja
    cmake
    meson
)

echo ":: Antenora toolchain bootstrap -> $SYSROOT (jobs=$JOBS)"
mkdir -p "$SYSROOT"

# seed the local recipe repository into the sysroot so `dante` can resolve
mkdir -p "$SYSROOT/var/lib/dante/repo"
cp -a "$REPO_DIR/recipes/." "$SYSROOT/var/lib/dante/repo/"

# use the prebuilt dante binary (avoids recompiling on every package)
DANTE="$REPO_DIR/dist/dante"
if [ ! -x "$DANTE" ]; then
    CGO_ENABLED=0 go -C "$REPO_DIR" build -trimpath -ldflags '-s -w' -o "$DANTE" ./cmd/dante
fi

for pkg in "${ORDER[@]}"; do
    recipe="$REPO_DIR/recipes/$pkg.hell"
    [ -f "$recipe" ] || { echo "!! missing recipe $recipe, skipping"; continue; }
    echo
    echo ":: Building $pkg ..."
    DANTE_ROOT="$SYSROOT" "$DANTE" install "$pkg" || {
        echo "!! $pkg failed to build"; exit 1
    }
done

echo
echo ":: Toolchain bootstrap complete."
echo ":: Add it to your PATH:  export PATH=$SYSROOT/usr/bin:\$PATH"
echo ":: Or install natively with: dante toolchain"
